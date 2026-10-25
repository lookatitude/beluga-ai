package nova

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"iter"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/google/uuid"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/voice/s2s"
)

const (
	defaultModel  = "amazon.nova-sonic-v1:0"
	defaultRegion = "us-east-1"
	defaultVoice  = "matthew"

	// defaultInputSampleRate is the PCM input rate Nova Sonic expects.
	defaultInputSampleRate = 16000
	// outputSampleRate is the PCM rate of Nova Sonic audio output.
	outputSampleRate = 24000

	// closeTimeout bounds the end-of-session events sent by Close.
	closeTimeout = 5 * time.Second
)

var _ s2s.S2S = (*Engine)(nil) // compile-time interface check

func init() {
	s2s.Register("nova", func(cfg s2s.Config) (s2s.S2S, error) {
		return New(cfg)
	})
}

// eventStream is the bidirectional byte stream a session runs over. The
// production implementation wraps Bedrock's InvokeModelWithBidirectionalStream
// event stream; tests substitute an in-memory fake.
type eventStream interface {
	// Send writes one JSON event.
	Send(ctx context.Context, payload []byte) error
	// Recv returns the next JSON event, or io.EOF when the stream ended
	// normally.
	Recv() ([]byte, error)
	// Close closes the stream.
	Close() error
}

// streamOpener opens a bidirectional stream for modelID.
type streamOpener func(ctx context.Context, modelID string) (eventStream, error)

// Engine implements s2s.S2S using Amazon Nova Sonic through the Bedrock
// Runtime InvokeModelWithBidirectionalStream API (HTTP/2, SigV4-signed by the
// AWS SDK).
type Engine struct {
	region string
	cfg    s2s.Config
	open   streamOpener
}

// New creates a new Amazon Nova Sonic S2S engine. AWS credentials come from
// Config.Extra ("access_key_id", "secret_access_key", optional
// "session_token") or, when absent, from the default AWS credential chain.
func New(cfg s2s.Config) (*Engine, error) {
	region, _ := cfg.Extra["region"].(string)
	if region == "" {
		region = defaultRegion
	}
	if cfg.Model == "" {
		cfg.Model = defaultModel
	}

	client, err := newBedrockClient(region, cfg.Extra)
	if err != nil {
		return nil, err
	}
	return newEngine(region, cfg, sdkOpener(client)), nil
}

func newEngine(region string, cfg s2s.Config, open streamOpener) *Engine {
	return &Engine{region: region, cfg: cfg, open: open}
}

// newBedrockClient builds a Bedrock Runtime client for region.
func newBedrockClient(region string, extra map[string]any) (*bedrockruntime.Client, error) {
	awsOpts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}
	if keyID, _ := extra["access_key_id"].(string); keyID != "" {
		secret, _ := extra["secret_access_key"].(string)
		token, _ := extra["session_token"].(string)
		awsOpts = append(awsOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(keyID, secret, token),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsOpts...)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "nova: load AWS config: %w", err)
	}

	var brOpts []func(*bedrockruntime.Options)
	if endpoint, _ := extra["base_url"].(string); endpoint != "" {
		if !strings.HasPrefix(endpoint, "https://") && !strings.HasPrefix(endpoint, "http://") {
			return nil, core.Errorf(core.ErrInvalidInput,
				"nova: base_url must be an http(s) Bedrock Runtime endpoint; the legacy WebSocket endpoint is not supported")
		}
		brOpts = append(brOpts, func(o *bedrockruntime.Options) { o.BaseEndpoint = aws.String(endpoint) })
	}
	return bedrockruntime.NewFromConfig(awsCfg, brOpts...), nil
}

// sdkOpener adapts a Bedrock Runtime client to a streamOpener.
func sdkOpener(client *bedrockruntime.Client) streamOpener {
	return func(ctx context.Context, modelID string) (eventStream, error) {
		out, err := client.InvokeModelWithBidirectionalStream(ctx, &bedrockruntime.InvokeModelWithBidirectionalStreamInput{
			ModelId: aws.String(modelID),
		})
		if err != nil {
			return nil, err
		}
		return &sdkStream{es: out.GetStream()}, nil
	}
}

// sdkStream wraps the SDK event stream as an eventStream.
type sdkStream struct {
	es *bedrockruntime.InvokeModelWithBidirectionalStreamEventStream
}

func (s *sdkStream) Send(ctx context.Context, payload []byte) error {
	return s.es.Send(ctx, &types.InvokeModelWithBidirectionalStreamInputMemberChunk{
		Value: types.BidirectionalInputPayloadPart{Bytes: payload},
	})
}

func (s *sdkStream) Recv() ([]byte, error) {
	for ev := range s.es.Events() {
		if chunk, ok := ev.(*types.InvokeModelWithBidirectionalStreamOutputMemberChunk); ok {
			return chunk.Value.Bytes, nil
		}
	}
	if err := s.es.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (s *sdkStream) Close() error { return s.es.Close() }

// Start opens a Nova Sonic bidirectional stream and sends the session
// preamble: sessionStart, promptStart (audio/text/tool output configuration
// and tools), the system prompt when Instructions is set, and the start of
// the interactive user audio content.
func (e *Engine) Start(ctx context.Context, opts ...s2s.Option) (s2s.Session, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	stream, err := e.open(ctx, cfg.Model)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "nova: open bidirectional stream: %w", err)
	}

	sess := &novaSession{
		stream:       stream,
		events:       make(chan s2s.SessionEvent, 64),
		done:         make(chan struct{}),
		cfg:          cfg,
		promptName:   uuid.NewString(),
		audioContent: uuid.NewString(),
		stages:       make(map[string]string),
	}
	if err := sess.sendPreamble(ctx); err != nil {
		_ = stream.Close()
		return nil, core.Errorf(core.ErrProviderDown, "nova: session setup: %w", err)
	}

	go sess.readLoop()
	return sess, nil
}

// novaSession implements s2s.Session for Amazon Nova Sonic.
type novaSession struct {
	stream eventStream
	events chan s2s.SessionEvent
	done   chan struct{}
	once   sync.Once
	cfg    s2s.Config

	promptName   string
	audioContent string

	sendMu sync.Mutex // serialises events so content blocks stay contiguous

	stageMu sync.Mutex
	stages  map[string]string // contentId -> generationStage
}

// send marshals {"event": {name: body}} and writes it to the stream.
func (s *novaSession) send(ctx context.Context, name string, body map[string]any) error {
	data, err := json.Marshal(map[string]any{"event": map[string]any{name: body}})
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "nova: encode %s: %w", name, err)
	}
	return s.stream.Send(ctx, data)
}

// sendLocked sends a sequence of events atomically with respect to other
// senders.
func (s *novaSession) sendLocked(ctx context.Context, events ...namedEvent) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	select {
	case <-s.done:
		return core.Errorf(core.ErrInvalidInput, "nova: session is closed")
	default:
	}
	for _, ev := range events {
		if err := s.send(ctx, ev.name, ev.body); err != nil {
			return err
		}
	}
	return nil
}

type namedEvent struct {
	name string
	body map[string]any
}

func (s *novaSession) sendPreamble(ctx context.Context) error {
	voiceID := s.cfg.Voice
	if voiceID == "" {
		voiceID = defaultVoice
	}
	inRate := s.cfg.SampleRate
	if inRate <= 0 {
		inRate = defaultInputSampleRate
	}

	promptStart := map[string]any{
		"promptName":              s.promptName,
		"textOutputConfiguration": map[string]any{"mediaType": "text/plain"},
		"audioOutputConfiguration": map[string]any{
			"mediaType":       "audio/lpcm",
			"sampleRateHertz": outputSampleRate,
			"sampleSizeBits":  16,
			"channelCount":    1,
			"voiceId":         voiceID,
			"encoding":        "base64",
			"audioType":       "SPEECH",
		},
	}
	if len(s.cfg.Tools) > 0 {
		tools := make([]map[string]any, 0, len(s.cfg.Tools))
		for _, t := range s.cfg.Tools {
			schemaJSON, err := json.Marshal(t.InputSchema)
			if err != nil {
				return core.Errorf(core.ErrInvalidInput, "nova: encode tool %q schema: %w", t.Name, err)
			}
			tools = append(tools, map[string]any{"toolSpec": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				// Nova Sonic takes the JSON schema as a JSON-encoded string.
				"inputSchema": map[string]any{"json": string(schemaJSON)},
			}})
		}
		promptStart["toolUseOutputConfiguration"] = map[string]any{"mediaType": "application/json"}
		promptStart["toolConfiguration"] = map[string]any{"tools": tools}
	}

	events := []namedEvent{
		{"sessionStart", map[string]any{"inferenceConfiguration": map[string]any{
			"maxTokens": 1024, "topP": 0.9, "temperature": 0.7,
		}}},
		{"promptStart", promptStart},
	}
	if s.cfg.Instructions != "" {
		events = append(events, s.textContent("SYSTEM", false, s.cfg.Instructions)...)
	}
	events = append(events, namedEvent{"contentStart", map[string]any{
		"promptName":  s.promptName,
		"contentName": s.audioContent,
		"type":        "AUDIO",
		"interactive": true,
		"role":        "USER",
		"audioInputConfiguration": map[string]any{
			"mediaType":       "audio/lpcm",
			"sampleRateHertz": inRate,
			"sampleSizeBits":  16,
			"channelCount":    1,
			"audioType":       "SPEECH",
			"encoding":        "base64",
		},
	}})
	return s.sendLocked(ctx, events...)
}

// textContent returns the contentStart/textInput/contentEnd triple for a
// text content block.
func (s *novaSession) textContent(role string, interactive bool, text string) []namedEvent {
	name := uuid.NewString()
	return []namedEvent{
		{"contentStart", map[string]any{
			"promptName":             s.promptName,
			"contentName":            name,
			"type":                   "TEXT",
			"interactive":            interactive,
			"role":                   role,
			"textInputConfiguration": map[string]any{"mediaType": "text/plain"},
		}},
		{"textInput", map[string]any{"promptName": s.promptName, "contentName": name, "content": text}},
		{"contentEnd", map[string]any{"promptName": s.promptName, "contentName": name}},
	}
}

// novaOutput is the union of Nova Sonic output events.
type novaOutput struct {
	Event struct {
		ContentStart *struct {
			ContentID             string `json:"contentId"`
			Role                  string `json:"role"`
			Type                  string `json:"type"`
			AdditionalModelFields string `json:"additionalModelFields"`
		} `json:"contentStart"`
		TextOutput *struct {
			ContentID string `json:"contentId"`
			Role      string `json:"role"`
			Content   string `json:"content"`
		} `json:"textOutput"`
		AudioOutput *struct {
			Content string `json:"content"`
		} `json:"audioOutput"`
		ToolUse *struct {
			ToolUseID string `json:"toolUseId"`
			ToolName  string `json:"toolName"`
			Content   string `json:"content"`
		} `json:"toolUse"`
		ContentEnd *struct {
			ContentID  string `json:"contentId"`
			StopReason string `json:"stopReason"`
			Type       string `json:"type"`
		} `json:"contentEnd"`
	} `json:"event"`
}

func (s *novaSession) readLoop() {
	defer close(s.events)
	for {
		data, err := s.stream.Recv()
		if err != nil {
			if !errors.Is(err, io.EOF) && !s.isClosed() {
				s.emit(s2s.SessionEvent{Type: s2s.EventError, Error: core.Errorf(core.ErrProviderDown, "nova: stream: %w", err)})
			}
			return
		}
		var out novaOutput
		if json.Unmarshal(data, &out) != nil {
			continue
		}
		for _, ev := range s.translate(out) {
			if !s.emit(ev) {
				return
			}
		}
	}
}

// translate maps one Nova output event to zero or more session events.
func (s *novaSession) translate(out novaOutput) []s2s.SessionEvent {
	e := out.Event
	switch {
	case e.ContentStart != nil:
		s.recordStage(e.ContentStart.ContentID, e.ContentStart.AdditionalModelFields)
	case e.TextOutput != nil:
		return s.translateText(e.TextOutput.ContentID, e.TextOutput.Role, e.TextOutput.Content)
	case e.AudioOutput != nil:
		audio, err := base64.StdEncoding.DecodeString(e.AudioOutput.Content)
		if err == nil && len(audio) > 0 {
			return []s2s.SessionEvent{{Type: s2s.EventAudioOutput, Audio: audio}}
		}
	case e.ToolUse != nil:
		return []s2s.SessionEvent{{Type: s2s.EventToolCall, ToolCall: &schema.ToolCall{
			ID:        e.ToolUse.ToolUseID,
			Name:      e.ToolUse.ToolName,
			Arguments: e.ToolUse.Content,
		}}}
	case e.ContentEnd != nil:
		s.stageMu.Lock()
		delete(s.stages, e.ContentEnd.ContentID)
		s.stageMu.Unlock()
		if e.ContentEnd.StopReason == "END_TURN" || e.ContentEnd.StopReason == "INTERRUPTED" {
			return []s2s.SessionEvent{{Type: s2s.EventTurnEnd}}
		}
	}
	return nil
}

func (s *novaSession) translateText(contentID, role, text string) []s2s.SessionEvent {
	if text == "" {
		return nil
	}
	if role == "USER" {
		return []s2s.SessionEvent{{Type: s2s.EventTranscript, Text: text}}
	}
	// Nova emits assistant text twice: a SPECULATIVE preview and the FINAL
	// transcript of what was spoken. Only the FINAL text is surfaced.
	s.stageMu.Lock()
	stage := s.stages[contentID]
	s.stageMu.Unlock()
	if stage == "SPECULATIVE" {
		return nil
	}
	return []s2s.SessionEvent{{Type: s2s.EventTextOutput, Text: text}}
}

func (s *novaSession) recordStage(contentID, fields string) {
	if contentID == "" || fields == "" {
		return
	}
	var f struct {
		GenerationStage string `json:"generationStage"`
	}
	if json.Unmarshal([]byte(fields), &f) != nil || f.GenerationStage == "" {
		return
	}
	s.stageMu.Lock()
	s.stages[contentID] = f.GenerationStage
	s.stageMu.Unlock()
}

// emit delivers ev unless the session was closed; it reports whether the
// read loop should continue.
func (s *novaSession) emit(ev s2s.SessionEvent) bool {
	select {
	case s.events <- ev:
		return true
	case <-s.done:
		return false
	}
}

func (s *novaSession) isClosed() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// SendAudio sends a PCM audio chunk on the interactive user audio content.
func (s *novaSession) SendAudio(ctx context.Context, audio []byte) error {
	return s.sendLocked(ctx, namedEvent{"audioInput", map[string]any{
		"promptName":  s.promptName,
		"contentName": s.audioContent,
		"content":     base64.StdEncoding.EncodeToString(audio),
	}})
}

// SendText sends a user text turn as an interactive TEXT content block.
func (s *novaSession) SendText(ctx context.Context, text string) error {
	return s.sendLocked(ctx, s.textContent("USER", true, text)...)
}

// SendToolResult sends a tool execution result back to Nova as a TOOL
// content block.
func (s *novaSession) SendToolResult(ctx context.Context, result schema.ToolResult) error {
	var output strings.Builder
	for _, part := range result.Content {
		if tp, ok := part.(schema.TextPart); ok {
			output.WriteString(tp.Text)
		}
	}
	name := uuid.NewString()
	return s.sendLocked(ctx,
		namedEvent{"contentStart", map[string]any{
			"promptName":  s.promptName,
			"contentName": name,
			"interactive": false,
			"type":        "TOOL",
			"role":        "TOOL",
			"toolResultInputConfiguration": map[string]any{
				"toolUseId":              result.CallID,
				"type":                   "TEXT",
				"textInputConfiguration": map[string]any{"mediaType": "text/plain"},
			},
		}},
		namedEvent{"toolResult", map[string]any{"promptName": s.promptName, "contentName": name, "content": output.String()}},
		namedEvent{"contentEnd", map[string]any{"promptName": s.promptName, "contentName": name}},
	)
}

// Recv returns an iterator over session events. The iterator ends when the
// stream ends, the session is closed, or ctx is cancelled.
func (s *novaSession) Recv(ctx context.Context) iter.Seq2[s2s.SessionEvent, error] {
	return func(yield func(s2s.SessionEvent, error) bool) {
		for {
			select {
			case event, ok := <-s.events:
				if !ok {
					return
				}
				if !yield(event, nil) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

// Interrupt is a no-op: Nova Sonic detects barge-in from the user audio
// stream itself and ends the assistant turn with stopReason INTERRUPTED,
// which surfaces as EventTurnEnd. There is no client-side interrupt event.
func (s *novaSession) Interrupt(_ context.Context) error {
	if s.isClosed() {
		return core.Errorf(core.ErrInvalidInput, "nova: session is closed")
	}
	return nil
}

// Close ends the audio content, the prompt and the session, then closes the
// stream. It is safe to call more than once.
func (s *novaSession) Close() error {
	var err error
	s.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		s.sendMu.Lock()
		// Best effort: the stream may already be broken.
		_ = s.send(ctx, "contentEnd", map[string]any{"promptName": s.promptName, "contentName": s.audioContent})
		_ = s.send(ctx, "promptEnd", map[string]any{"promptName": s.promptName})
		_ = s.send(ctx, "sessionEnd", map[string]any{})
		close(s.done)
		s.sendMu.Unlock()
		err = s.stream.Close()
	})
	return err
}
