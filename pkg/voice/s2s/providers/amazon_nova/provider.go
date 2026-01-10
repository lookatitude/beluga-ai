package amazon_nova

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// AmazonNovaProvider implements the S2SProvider interface for Amazon Nova 2 Sonic.
type AmazonNovaProvider struct {
	config       *AmazonNovaConfig
	client       *bedrockruntime.Client
	mu           sync.RWMutex
	providerName string
}

// NewAmazonNovaProvider creates a new Amazon Nova 2 Sonic provider.
func NewAmazonNovaProvider(config *s2s.Config) (iface.S2SProvider, error) {
	if config == nil {
		return nil, s2s.NewS2SError("NewAmazonNovaProvider", s2s.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Amazon Nova config
	novaConfig := &AmazonNovaConfig{
		Config: config,
	}

	// Set defaults if not provided
	if novaConfig.Region == "" {
		novaConfig.Region = "us-east-1"
	}
	if novaConfig.Model == "" {
		novaConfig.Model = "nova-2-sonic"
	}
	if novaConfig.VoiceID == "" {
		novaConfig.VoiceID = "Ruth"
	}
	if novaConfig.LanguageCode == "" {
		novaConfig.LanguageCode = "en-US"
	}
	if novaConfig.Timeout == 0 {
		novaConfig.Timeout = 30 * time.Second
	}
	if novaConfig.SampleRate == 0 {
		novaConfig.SampleRate = 24000
	}
	if novaConfig.AudioFormat == "" {
		novaConfig.AudioFormat = "pcm"
	}

	// Load AWS configuration
	ctx := context.Background()
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(novaConfig.Region),
	)
	if err != nil {
		return nil, s2s.NewS2SError("NewAmazonNovaProvider", s2s.ErrCodeInvalidConfig,
			fmt.Errorf("failed to load AWS config: %w", err))
	}

	// Create Bedrock Runtime client
	clientOptions := []func(*bedrockruntime.Options){}
	if novaConfig.EndpointURL != "" {
		clientOptions = append(clientOptions, func(o *bedrockruntime.Options) {
			o.BaseEndpoint = aws.String(novaConfig.EndpointURL)
		})
	}
	client := bedrockruntime.NewFromConfig(awsCfg, clientOptions...)

	return &AmazonNovaProvider{
		config:       novaConfig,
		client:       client,
		providerName: "amazon_nova",
	}, nil
}

// Process implements the S2SProvider interface using Amazon Nova 2 Sonic API.
func (p *AmazonNovaProvider) Process(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
	startTime := time.Now()

	// Start tracing
	ctx, span := s2s.StartProcessSpan(ctx, p.providerName, p.config.Model, input.Language)
	defer span.End()

	// Validate input
	if err := internal.ValidateAudioInput(input); err != nil {
		s2s.RecordSpanError(span, err)
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidInput, err)
	}

	// Apply options
	stsOpts := &internal.STSOptions{}
	for _, opt := range opts {
		opt(stsOpts)
	}

	// Prepare the request body for Bedrock Runtime API
	// Nova 2 Sonic uses a specific request format
	requestBody, err := p.prepareNovaRequest(input, convCtx, stsOpts)
	if err != nil {
		s2s.RecordSpanError(span, err)
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidRequest,
			fmt.Errorf("failed to prepare request: %w", err))
	}

	// Set timeout context
	requestCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// Call Bedrock Runtime API
	modelID := fmt.Sprintf("amazon.%s-v1:0", p.config.Model)
	invokeInput := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Body:        requestBody,
	}

	invokeOutput, err := p.client.InvokeModel(requestCtx, invokeInput)
	if err != nil {
		s2s.RecordSpanError(span, err)
		s2s.RecordSpanLatency(span, time.Since(startTime))
		return nil, p.handleBedrockError("Process", err)
	}

	// Parse response
	output, err := p.parseNovaResponse(invokeOutput.Body, input)
	if err != nil {
		s2s.RecordSpanError(span, err)
		s2s.RecordSpanLatency(span, time.Since(startTime))
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidResponse,
			fmt.Errorf("failed to parse response: %w", err))
	}

	output.Latency = time.Since(startTime)
	s2s.RecordSpanLatency(span, output.Latency)
	s2s.RecordSpanAttributes(span, map[string]string{
		"output_size": fmt.Sprintf("%d", len(output.Data)),
		"latency_ms":  fmt.Sprintf("%d", output.Latency.Milliseconds()),
	})

	return output, nil
}

// prepareNovaRequest prepares the request body for Nova 2 Sonic API.
func (p *AmazonNovaProvider) prepareNovaRequest(input *internal.AudioInput, convCtx *internal.ConversationContext, opts *internal.STSOptions) ([]byte, error) {
	// Encode audio data as base64
	audioBase64 := base64.StdEncoding.EncodeToString(input.Data)

	// Build request payload
	request := map[string]any{
		"input": map[string]any{
			"audio": audioBase64,
			"format": map[string]any{
				"sample_rate": input.Format.SampleRate,
				"channels":      input.Format.Channels,
				"encoding":    input.Format.Encoding,
			},
		},
		"voice": map[string]any{
			"voice_id": p.config.VoiceID,
			"language": p.config.LanguageCode,
		},
		"output": map[string]any{
			"format": map[string]any{
				"sample_rate": p.config.SampleRate,
				"channels":    input.Format.Channels,
				"encoding":    p.config.AudioFormat,
			},
		},
	}

	// Add conversation context if provided
	if convCtx != nil {
		if convCtx.ConversationID != "" {
			request["conversation_id"] = convCtx.ConversationID
		}
		if len(convCtx.History) > 0 {
			history := make([]map[string]any, 0, len(convCtx.History))
			for _, turn := range convCtx.History {
				history = append(history, map[string]any{
					"role":      turn.Role,
					"content":   turn.Content,
					"timestamp": turn.Timestamp.Unix(),
				})
			}
			request["conversation_history"] = history
		}
	}

	// Add options
	if opts != nil {
		if opts.Language != "" {
			request["voice"].(map[string]any)["language"] = opts.Language
		}
		if opts.VoiceID != "" {
			request["voice"].(map[string]any)["voice_id"] = opts.VoiceID
		}
	}

	// Add automatic punctuation setting
	request["enable_automatic_punctuation"] = p.config.EnableAutomaticPunctuation

	return json.Marshal(request)
}

// parseNovaResponse parses the response from Nova 2 Sonic API.
func (p *AmazonNovaProvider) parseNovaResponse(responseBody []byte, input *internal.AudioInput) (*internal.AudioOutput, error) {
	var response struct {
		Output struct {
			Audio  string `json:"audio"`
			Format struct {
				SampleRate int    `json:"sample_rate"`
				Channels   int    `json:"channels"`
				Encoding   string `json:"encoding"`
			} `json:"format"`
		} `json:"output"`
		Voice struct {
			VoiceID  string `json:"voice_id"`
			Language string `json:"language"`
		} `json:"voice"`
		Metadata struct {
			LatencyMs float64 `json:"latency_ms"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(response.Output.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %w", err)
	}

	output := &internal.AudioOutput{
		Data: audioData,
		Format: internal.AudioFormat{
			SampleRate: response.Output.Format.SampleRate,
			Channels:   response.Output.Format.Channels,
			BitDepth:   input.Format.BitDepth,
			Encoding:   response.Output.Format.Encoding,
		},
		Timestamp: time.Now(),
		Provider:  p.providerName,
		VoiceCharacteristics: internal.VoiceCharacteristics{
			VoiceID:  response.Voice.VoiceID,
			Language: response.Voice.Language,
		},
	}

	return output, nil
}

// handleBedrockError handles errors from Bedrock Runtime API.
func (p *AmazonNovaProvider) handleBedrockError(operation string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "ThrottlingException") || strings.Contains(errStr, "429") {
		errorCode = s2s.ErrCodeRateLimit
		message = "Amazon Nova API rate limit exceeded"
	} else if strings.Contains(errStr, "ValidationException") {
		errorCode = s2s.ErrCodeInvalidRequest
		message = "Amazon Nova API validation failed"
	} else if strings.Contains(errStr, "ModelNotReadyException") {
		errorCode = s2s.ErrCodeModelNotAvailable
		message = "Amazon Nova model not available"
	} else {
		errorCode = s2s.ErrCodeInvalidRequest
		message = "Amazon Nova API request failed"
	}

	return s2s.NewS2SErrorWithMessage(operation, errorCode, message, err)
}

// Name implements the S2SProvider interface.
func (p *AmazonNovaProvider) Name() string {
	return p.providerName
}
