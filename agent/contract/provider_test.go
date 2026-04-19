package contract

import (
	"context"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
	"github.com/stretchr/testify/assert"
)

// agentWithContract implements ContractProvider.
type agentWithContract struct {
	contract *schema.Contract
}

var _ agent.Agent = (*agentWithContract)(nil)

func (a *agentWithContract) ID() string                 { return "with-contract" }
func (a *agentWithContract) Persona() agent.Persona     { return agent.Persona{} }
func (a *agentWithContract) Tools() []tool.Tool         { return nil }
func (a *agentWithContract) Children() []agent.Agent    { return nil }
func (a *agentWithContract) Contract() *schema.Contract { return a.contract }

func (a *agentWithContract) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	return "", nil
}

func (a *agentWithContract) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {}
}

// agentWithoutContract does not implement ContractProvider.
type agentWithoutContract struct{}

var _ agent.Agent = (*agentWithoutContract)(nil)

func (a *agentWithoutContract) ID() string              { return "no-contract" }
func (a *agentWithoutContract) Persona() agent.Persona  { return agent.Persona{} }
func (a *agentWithoutContract) Tools() []tool.Tool      { return nil }
func (a *agentWithoutContract) Children() []agent.Agent { return nil }

func (a *agentWithoutContract) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	return "", nil
}

func (a *agentWithoutContract) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {}
}

func TestContractOf_WithProvider(t *testing.T) {
	c := &schema.Contract{Name: "test"}
	a := &agentWithContract{contract: c}

	got := ContractOf(a)
	assert.Equal(t, c, got)
}

func TestContractOf_WithoutProvider(t *testing.T) {
	a := &agentWithoutContract{}

	got := ContractOf(a)
	assert.Nil(t, got)
}

func TestContractOf_NilContract(t *testing.T) {
	a := &agentWithContract{contract: nil}

	got := ContractOf(a)
	assert.Nil(t, got)
}
