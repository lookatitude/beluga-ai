package assemblyai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/lookatitude/beluga-ai/v2/voice/stt"
)

const (
	defaultBaseURL = "https://api.assemblyai.com/v2"
	// defaultWSURL is the Universal-Streaming (v3) endpoint. The v2
	// real-time endpoint (wss://api.assemblyai.com/v2/realtime/ws) is retired.
	defaultWSURL = "wss://streaming.assemblyai.com/v3/ws"
)

var _ stt.STT = (*Engine)(nil) // compile-time interface check

func init() {
	stt.Register("assemblyai", func(cfg stt.Config) (stt.STT, error) {
		return New(cfg)
	})
}

// Engine implements stt.STT using the AssemblyAI API.
type Engine struct {
	apiKey  string
	baseURL string
	wsURL   string
	cfg     stt.Config
}

// New creates a new AssemblyAI STT engine.
func New(cfg stt.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("assemblyai: api_key is required in Extra")
	}

	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	wsURL, _ := cfg.Extra["ws_url"].(string)
	if wsURL == "" {
		wsURL = defaultWSURL
	}

	return &Engine{
		apiKey:  apiKey,
		baseURL: baseURL,
		wsURL:   wsURL,
		cfg:     cfg,
	}, nil
}

// uploadResponse is the response from the AssemblyAI upload endpoint.
type uploadResponse struct {
	UploadURL string `json:"upload_url"`
}

// transcriptRequest is the request body for creating a transcript.
type transcriptRequest struct {
	AudioURL      string `json:"audio_url"`
	LanguageCode  string `json:"language_code,omitempty"`
	Punctuate     *bool  `json:"punctuate,omitempty"`
	SpeakerLabels *bool  `json:"speaker_labels,omitempty"`
}

// transcriptResponse is the response from the transcript endpoint.
type transcriptResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Text   string `json:"text"`
	Words  []struct {
		Text       string  `json:"text"`
		Start      int     `json:"start"`
		End        int     `json:"end"`
		Confidence float64 `json:"confidence"`
	} `json:"words"`
	Error string `json:"error,omitempty"`
}

// uploadAudio uploads audio and returns the upload URL.
func (e *Engine) uploadAudio(ctx context.Context, audio []byte) (string, error) {
	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/upload", bytes.NewReader(audio))
	if err != nil {
		return "", fmt.Errorf("assemblyai: create upload request: %w", err)
	}
	uploadReq.Header.Set("Authorization", e.apiKey)
	uploadReq.Header.Set("Content-Type", "application/octet-stream")

	uploadResp, err := http.DefaultClient.Do(uploadReq)
	if err != nil {
		return "", fmt.Errorf("assemblyai: upload failed: %w", err)
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(uploadResp.Body)
		return "", fmt.Errorf("assemblyai: upload error (status %d): %s", uploadResp.StatusCode, body)
	}

	var upload uploadResponse
	if err := json.NewDecoder(uploadResp.Body).Decode(&upload); err != nil {
		return "", fmt.Errorf("assemblyai: decode upload response: %w", err)
	}
	return upload.UploadURL, nil
}

// createTranscript creates a transcript job and returns the initial response.
func (e *Engine) createTranscript(ctx context.Context, audioURL string, cfg stt.Config) (transcriptResponse, error) {
	txReq := transcriptRequest{
		AudioURL: audioURL,
	}
	if cfg.Language != "" {
		txReq.LanguageCode = cfg.Language
	}
	if cfg.Punctuation {
		v := true
		txReq.Punctuate = &v
	}
	if cfg.Diarization {
		v := true
		txReq.SpeakerLabels = &v
	}

	txData, err := json.Marshal(txReq)
	if err != nil {
		return transcriptResponse{}, fmt.Errorf("assemblyai: marshal request: %w", err)
	}

	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/transcript", bytes.NewReader(txData))
	if err != nil {
		return transcriptResponse{}, fmt.Errorf("assemblyai: create transcript request: %w", err)
	}
	createReq.Header.Set("Authorization", e.apiKey)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		return transcriptResponse{}, fmt.Errorf("assemblyai: create transcript failed: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(createResp.Body)
		return transcriptResponse{}, fmt.Errorf("assemblyai: create error (status %d): %s", createResp.StatusCode, body)
	}

	var txResult transcriptResponse
	if err := json.NewDecoder(createResp.Body).Decode(&txResult); err != nil {
		return transcriptResponse{}, fmt.Errorf("assemblyai: decode transcript response: %w", err)
	}
	return txResult, nil
}

// pollTranscript polls for transcript completion and returns the final text.
func (e *Engine) pollTranscript(ctx context.Context, txResult transcriptResponse) (string, error) {
	for txResult.Status != "completed" && txResult.Status != "error" {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}

		pollReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
			e.baseURL+"/transcript/"+txResult.ID, nil)
		if err != nil {
			return "", fmt.Errorf("assemblyai: create poll request: %w", err)
		}
		pollReq.Header.Set("Authorization", e.apiKey)

		pollResp, err := http.DefaultClient.Do(pollReq)
		if err != nil {
			return "", fmt.Errorf("assemblyai: poll failed: %w", err)
		}

		if err := json.NewDecoder(pollResp.Body).Decode(&txResult); err != nil {
			_ = pollResp.Body.Close()
			return "", fmt.Errorf("assemblyai: decode poll response: %w", err)
		}
		_ = pollResp.Body.Close()
	}

	if txResult.Status == "error" {
		return "", fmt.Errorf("assemblyai: transcription failed: %s", txResult.Error)
	}
	return txResult.Text, nil
}

// Transcribe converts audio to text using AssemblyAI's REST API.
func (e *Engine) Transcribe(ctx context.Context, audio []byte, opts ...stt.Option) (string, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	uploadURL, err := e.uploadAudio(ctx, audio)
	if err != nil {
		return "", err
	}

	txResult, err := e.createTranscript(ctx, uploadURL, cfg)
	if err != nil {
		return "", err
	}

	return e.pollTranscript(ctx, txResult)
}

// streamWord is a word in a v3 Turn message. Times are milliseconds from the
// start of the audio stream.
type streamWord struct {
	Text        string  `json:"text"`
	Start       int     `json:"start"`
	End         int     `json:"end"`
	Confidence  float64 `json:"confidence"`
	WordIsFinal bool    `json:"word_is_final"`
}

// turnMessage is a Universal-Streaming (v3) server message. Only "Turn"
// messages carry transcripts; "Begin" and "Termination" are ignored.
type turnMessage struct {
	Type            string       `json:"type"`
	Transcript      string       `json:"transcript"`
	EndOfTurn       bool         `json:"end_of_turn"`
	TurnIsFormatted bool         `json:"turn_is_formatted"`
	Words           []streamWord `json:"words"`
}

// streamSpeechModel picks the v3 speech_model: cfg.Model when set, otherwise
// the English model for empty/English languages and the multilingual model
// for anything else.
func streamSpeechModel(cfg stt.Config) string {
	if cfg.Model != "" {
		return cfg.Model
	}
	lang := strings.ToLower(cfg.Language)
	if lang == "" || lang == "en" || strings.HasPrefix(lang, "en-") {
		return "universal-streaming-english"
	}
	return "universal-streaming-multilingual"
}

// dialStream opens a WebSocket connection to AssemblyAI's Universal-Streaming
// (v3) endpoint. The API key goes in the Authorization header (no Bearer
// prefix), never in the URL.
func (e *Engine) dialStream(ctx context.Context, cfg stt.Config) (*websocket.Conn, error) {
	rate := cfg.SampleRate
	if rate <= 0 {
		rate = 16000
	}
	q := url.Values{}
	q.Set("sample_rate", strconv.Itoa(rate))
	q.Set("encoding", "pcm_s16le")
	q.Set("format_turns", "true")
	q.Set("speech_model", streamSpeechModel(cfg))
	wsEndpoint := e.wsURL + "?" + q.Encode()

	headers := http.Header{}
	headers.Set("Authorization", e.apiKey)

	conn, _, err := websocket.Dial(ctx, wsEndpoint, &websocket.DialOptions{ //nolint:bodyclose // coder/websocket nils resp.Body on success and closes it on error
		HTTPHeader: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("assemblyai: websocket dial: %w", err)
	}
	return conn, nil
}

// parseStreamMessage parses a v3 Turn message into a TranscriptEvent.
// Returns the event and whether it should be emitted. A turn is final once
// it has ended and the formatted transcript has arrived (format_turns=true).
func (e *Engine) parseStreamMessage(data []byte, language string) (stt.TranscriptEvent, bool) {
	var msg turnMessage
	if json.Unmarshal(data, &msg) != nil || msg.Type != "Turn" || msg.Transcript == "" {
		return stt.TranscriptEvent{}, false
	}

	var (
		words []stt.Word
		sum   float64
	)
	for _, w := range msg.Words {
		words = append(words, stt.Word{
			Text:       w.Text,
			Start:      time.Duration(w.Start) * time.Millisecond,
			End:        time.Duration(w.End) * time.Millisecond,
			Confidence: w.Confidence,
		})
		sum += w.Confidence
	}
	ev := stt.TranscriptEvent{
		Text:     msg.Transcript,
		IsFinal:  msg.EndOfTurn && msg.TurnIsFormatted,
		Language: language,
		Words:    words,
	}
	if len(msg.Words) > 0 {
		ev.Confidence = sum / float64(len(msg.Words))
		ev.Timestamp = time.Duration(msg.Words[0].Start) * time.Millisecond
	}
	return ev, true
}

// readStreamMessages reads WebSocket messages and sends parsed events to the channel.
func (e *Engine) readStreamMessages(ctx context.Context, conn *websocket.Conn, language string, events chan<- stt.TranscriptEvent, errs chan<- error) {
	defer close(events)
	for {
		_, data, readErr := conn.Read(ctx)
		if readErr != nil {
			if ctx.Err() == nil && websocket.CloseStatus(readErr) != websocket.StatusNormalClosure {
				select {
				case errs <- fmt.Errorf("assemblyai: websocket read: %w", readErr):
				default:
				}
			}
			return
		}
		if event, ok := e.parseStreamMessage(data, language); ok {
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}
}

// sendAudioStream sends audio chunks as binary messages and then the v3
// Terminate message.
func (e *Engine) sendAudioStream(ctx context.Context, conn *websocket.Conn, audioStream iter.Seq2[[]byte, error], errs chan<- error) {
	for chunk, streamErr := range audioStream {
		if streamErr != nil {
			errs <- streamErr
			return
		}
		if writeErr := conn.Write(ctx, websocket.MessageBinary, chunk); writeErr != nil {
			if ctx.Err() == nil {
				select {
				case errs <- fmt.Errorf("assemblyai: websocket write: %w", writeErr):
				default:
				}
			}
			return
		}
	}
	// Signal end of stream.
	_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"Terminate"}`)) // #nosec G104 -- best-effort terminate; connection cleanup continues regardless
}

// TranscribeStream converts streaming audio to transcript events
// using AssemblyAI's Universal-Streaming (v3) WebSocket API.
func (e *Engine) TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...stt.Option) iter.Seq2[stt.TranscriptEvent, error] {
	return func(yield func(stt.TranscriptEvent, error) bool) {
		cfg := e.cfg
		for _, opt := range opts {
			opt(&cfg)
		}

		conn, err := e.dialStream(ctx, cfg)
		if err != nil {
			yield(stt.TranscriptEvent{}, err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		events := make(chan stt.TranscriptEvent, 16)
		errs := make(chan error, 1)

		go e.readStreamMessages(ctx, conn, cfg.Language, events, errs)
		go e.sendAudioStream(ctx, conn, audioStream, errs)

		drainStreamEvents(ctx, events, errs, yield)
	}
}

// drainStreamEvents reads from event and error channels, yielding to the caller.
func drainStreamEvents(ctx context.Context, events <-chan stt.TranscriptEvent, errs <-chan error, yield func(stt.TranscriptEvent, error) bool) {
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if !yield(event, nil) {
				return
			}
		case err := <-errs:
			yield(stt.TranscriptEvent{}, err)
			return
		case <-ctx.Done():
			yield(stt.TranscriptEvent{}, ctx.Err())
			return
		}
	}
}
