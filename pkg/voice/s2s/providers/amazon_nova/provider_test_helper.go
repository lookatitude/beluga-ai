package amazon_nova

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
)

// NewAmazonNovaProviderWithClient creates a new Amazon Nova provider with a custom Bedrock client.
// This is useful for testing with mock Bedrock clients.
// Note: Currently, the provider uses a concrete *bedrockruntime.Client type, so proper mocking
// would require refactoring the provider to use the BedrockRuntimeClient interface.
func NewAmazonNovaProviderWithClient(config *s2s.Config, client *bedrockruntime.Client) (iface.S2SProvider, error) {
	if config == nil {
		return nil, s2s.NewS2SError("NewAmazonNovaProviderWithClient", s2s.ErrCodeInvalidConfig,
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

	// Use provided client or create a real one
	var bedrockClient *bedrockruntime.Client
	if client != nil {
		bedrockClient = client
	} else {
		// Load AWS configuration
		ctx := context.Background()
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(novaConfig.Region),
		)
		if err != nil {
			return nil, s2s.NewS2SError("NewAmazonNovaProviderWithClient", s2s.ErrCodeInvalidConfig,
				fmt.Errorf("failed to load AWS config: %w", err))
		}

		// Create Bedrock Runtime client
		clientOptions := []func(*bedrockruntime.Options){}
		if novaConfig.EndpointURL != "" {
			clientOptions = append(clientOptions, func(o *bedrockruntime.Options) {
				o.BaseEndpoint = aws.String(novaConfig.EndpointURL)
			})
		}
		bedrockClient = bedrockruntime.NewFromConfig(awsCfg, clientOptions...)
	}

	return &AmazonNovaProvider{
		config:       novaConfig,
		client:       bedrockClient,
		providerName: "amazon_nova",
	}, nil
}
