package mock

import (
	"context"
)

// Executor is a mock implementation of the command executor interface.
// It implements the same interface as mcp.Executor through structural typing.
type Executor struct {
	ExecuteInvoked bool
	Dir            string
	Name           string
	Args           []string
	ExecuteFn      func(context.Context, string, string, ...string) ([]byte, error)
}

// NewExecutor creates a new mock executor
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute implements the executor interface, recording invocation details
// and delegating to ExecuteFn if provided.
func (m *Executor) Execute(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	m.ExecuteInvoked = true
	m.Dir = dir
	m.Name = name
	m.Args = args

	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, dir, name, args...)
	}

	// Default behavior: return empty success
	return []byte(""), nil
}
