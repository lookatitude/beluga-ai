// Package nova provides the Amazon Nova S2S provider for the Beluga AI voice
// pipeline. It uses the AWS Bedrock Runtime API for bidirectional audio
// streaming with Amazon Nova Sonic.
//
// # Registration
//
// This package registers itself as "nova" with the s2s registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/nova"
//
// # Usage
//
//	engine, err := s2s.New("nova", s2s.Config{
//	    Model: "amazon.nova-sonic-v1:0",
//	    Extra: map[string]any{"region": "us-east-1"},
//	})
//	session, err := engine.Start(ctx)
//	defer session.Close()
//
// # Configuration
//
// Configuration in Config.Extra:
//
//   - region — AWS region (optional, defaults to "us-east-1")
//   - base_url — Custom WebSocket endpoint (optional, defaults to Bedrock Runtime URL)
//
// The default model is "amazon.nova-sonic-v1:0". Instructions and tools are
// passed through [s2s.Config] fields.
//
// # Exported Types
//
//   - [Engine] — implements s2s.S2S using Amazon Nova via Bedrock
//   - [New] — constructor accepting s2s.Config
package nova
