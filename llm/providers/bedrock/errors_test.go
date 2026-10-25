package bedrock

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

func codeOf(err error) core.ErrorCode {
	var ce *core.Error
	if errors.As(err, &ce) {
		return ce.Code
	}
	return ""
}

type statusErr struct{ code int }

func (e *statusErr) Error() string       { return "http response error" }
func (e *statusErr) HTTPStatusCode() int { return e.code }

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want core.ErrorCode
	}{
		{"throttling", &brtypes.ThrottlingException{Message: aws.String("slow")}, core.ErrRateLimit},
		{"access denied", &brtypes.AccessDeniedException{Message: aws.String("no")}, core.ErrAuth},
		{"validation", &brtypes.ValidationException{Message: aws.String("bad")}, core.ErrInvalidInput},
		{"unavailable", &brtypes.ServiceUnavailableException{Message: aws.String("down")}, core.ErrProviderDown},
		{"model timeout", &brtypes.ModelTimeoutException{Message: aws.String("t")}, core.ErrTimeout},
		{"http status wins", fmt.Errorf("op: %w", &statusErr{code: 401}), core.ErrAuth},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeOf(classifyError("bedrock: converse failed", tt.err)); got != tt.want {
				t.Fatalf("code = %q, want %q", got, tt.want)
			}
		})
	}
	if core.IsRetryable(classifyError("bedrock", errors.New("unknown"))) {
		t.Fatal("unknown error must not be retryable")
	}
}

func TestGenerateAndStream_ClassifyErrors(t *testing.T) {
	client := &mockClient{
		converseFunc: func(context.Context, *bedrockruntime.ConverseInput, ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return nil, &brtypes.AccessDeniedException{Message: aws.String("no")}
		},
		converseStreamFunc: func(context.Context, *bedrockruntime.ConverseStreamInput, ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error) {
			return nil, &brtypes.ThrottlingException{Message: aws.String("slow")}
		},
	}
	m := NewWithClient(client, "test-model")
	msgs := []schema.Message{schema.NewHumanMessage("Hi")}
	_, err := m.Generate(context.Background(), msgs)
	if codeOf(err) != core.ErrAuth || core.IsRetryable(err) {
		t.Fatalf("Generate: want non-retryable auth, got %v", err)
	}
	var streamErr error
	for _, err := range m.Stream(context.Background(), msgs) {
		streamErr = err
	}
	if codeOf(streamErr) != core.ErrRateLimit {
		t.Fatalf("Stream: want rate limit, got %v", streamErr)
	}
}
