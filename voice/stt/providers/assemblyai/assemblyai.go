package assemblyai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/lookatitude/beluga-ai/voice/stt"
)

const (
	defaultBaseURL = "https://api.assemblyai.com/v2"
	defaultWSURL   = "wss://api.assemblyai.com/v2/realtime/ws"
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
	AudioURL    string `json:"audio_url"`
	LanguageCode string `json:"language_code,omitempty"`
	Punctuate    *bool  `json:"punctuate,omitempty"`
	SpeakerLabels *bool `json:"speaker_labels,omitempty"`
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

// Transcribe converts audio to text using AssemblyAI's REST API.
func (e *Engine) Transcribe(ctx context.Context, audio []byte, opts ...stt.Option) (string, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	// Upload audio.
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

	// Create transcript.
	txReq := transcriptRequest{
		AudioURL: upload.UploadURL,
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
		return "", fmt.Errorf("assemblyai: marshal request: %w", err)
	}

	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/transcript", bytes.NewReader(txData))
	if err != nil {
		return "", fmt.Errorf("assemblyai: create transcript request: %w", err)
	}
	createReq.Header.Set("Authorization", e.apiKey)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		return "", fmt.Errorf("assemblyai: create transcript failed: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(createResp.Body)
		return "", fmt.Errorf("assemblyai: create error (status %d): %s", createResp.StatusCode, body)
	}

	var txResult transcriptResponse
	if err := json.NewDecoder(createResp.Body).Decode(&txResult); err != nil {
		return "", fmt.Errorf("assemblyai: decode transcript response: %w", err)
	}

	// Poll for completion.
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
			pollResp.Body.Close()
			return "", fmt.Errorf("assemblyai: decode poll response: %w", err)
		}
		pollResp.Body.Close()
	}

	if txResult.Status == "error" {
		return "", fmt.Errorf("assemblyai: transcription failed: %s", txResult.Error)
	}

	return txResult.Text, nil
}

// realtimeMessage is a message from the AssemblyAI real-time WebSocket API.
type realtimeMessage struct {
	MessageType string `json:"message_type"`
	Text        string `json:"text"`
	AudioStart  int    `json:"audio_start"`
	AudioEnd    int    `json:"audio_end"`
	Confidence  float64 `json:"confidence"`
	Words       []struct {
		Text       string  `json:"text"`
		Start      int     `json:"start"`
		End        int     `json:"end"`
		Confidence float64 `json:"confidence"`
	} `json:"words"`
}

// TranscribeStream converts streaming audio to transcript events
// using AssemblyAI's real-time WebSocket API.
func (e *Engine) TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...stt.Option) iter.Seq2[stt.TranscriptEvent, error] {
	return func(yield func(stt.TranscriptEvent, error) bool) {
		cfg := e.cfg
		for _, opt := range opts {
			opt(&cfg)
		}

		wsEndpoint := e.wsURL + "?sample_rate=16000"
		if cfg.SampleRate > 0 {
			wsEndpoint = fmt.Sprintf("%s?sample_rate=%d", e.wsURL, cfg.SampleRate)
		}

		headers := http.Header{}
		headers.Set("Authorization", e.apiKey)

		conn, _, err := websocket.Dial(ctx, wsEndpoint, &websocket.DialOptions{
			HTTPHeader: headers,
		})
		if err != nil {
			yield(stt.TranscriptEvent{}, fmt.Errorf("assemblyai: websocket dial: %w", err))
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		events := make(chan stt.TranscriptEvent, 16)
		errs := make(chan error, 1)

		// Read events.
		go func() {
			defer close(events)
			for {
				_, data, readErr := conn.Read(ctx)
				if readErr != nil {
					if ctx.Err() == nil {
						errs <- fmt.Errorf("assemblyai: websocket read: %w", readErr)
					}
					return
				}

				var msg realtimeMessage
				if jsonErr := json.Unmarshal(data, &msg); jsonErr != nil {
					continue
				}

				if msg.Text == "" {
					continue
				}

				isFinal := msg.MessageType == "FinalTranscript"

				var words []stt.Word
				for _, w := range msg.Words {
					words = append(words, stt.Word{
						Text:       w.Text,
						Start:      time.Duration(w.Start) * time.Millisecond,
						End:        time.Duration(w.End) * time.Millisecond,
						Confidence: w.Confidence,
					})
				}

				events <- stt.TranscriptEvent{
					Text:       msg.Text,
					IsFinal:    isFinal,
					Confidence: msg.Confidence,
					Timestamp:  time.Duration(msg.AudioStart) * time.Millisecond,
					Language:   cfg.Language,
					Words:      words,
				}
			}
		}()

		// Send audio chunks.
		go func() {
			for chunk, streamErr := range audioStream {
				if streamErr != nil {
					errs <- streamErr
					return
				}
				msg := map[string]any{"audio_data": chunk}
				data, _ := json.Marshal(msg)
				if writeErr := conn.Write(ctx, websocket.MessageText, data); writeErr != nil {
					if ctx.Err() == nil {
						errs <- fmt.Errorf("assemblyai: websocket write: %w", writeErr)
					}
					return
				}
			}
			// Signal end of stream.
			conn.Write(ctx, websocket.MessageText, []byte(`{"terminate_session": true}`))
		}()

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
}
