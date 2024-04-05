package mock

import (
	"context"
	"errors"

	fn "knative.dev/func/pkg/functions"
)

type PipelinesProvider struct {
	RunInvoked          bool
	RunFn               func(fn.Function) (string, string, error)
	RemoveInvoked       bool
	RemoveFn            func(fn.Function) error
	ConfigurePACInvoked bool
	ConfigurePACFn      func(fn.Function) error
	RemovePACInvoked    bool
	RemovePACFn         func(fn.Function) error
}

func NewPipelinesProvider() *PipelinesProvider {
	return &PipelinesProvider{
		RunFn: func(f fn.Function) (x string, namespace string, err error) {
			// the minimum necessary logic for a deployer, which should be
			// confirmed by tests in the respective implementations.
			if f.Namespace != "" {
				namespace = f.Namespace
			} else if f.Deploy.Namespace != "" {
				namespace = f.Deploy.Namespace
			} else {
				err = errors.New("namespace required for initial deployment")
			}
			return
		},
		RemoveFn:       func(fn.Function) error { return nil },
		ConfigurePACFn: func(fn.Function) error { return nil },
		RemovePACFn:    func(fn.Function) error { return nil },
	}
}

func (p *PipelinesProvider) Run(ctx context.Context, f fn.Function) (string, string, error) {
	p.RunInvoked = true
	return p.RunFn(f)
}

func (p *PipelinesProvider) Remove(ctx context.Context, f fn.Function) error {
	p.RemoveInvoked = true
	return p.RemoveFn(f)
}

func (p *PipelinesProvider) ConfigurePAC(ctx context.Context, f fn.Function, metadata any) error {
	p.ConfigurePACInvoked = true
	return p.ConfigurePACFn(f)
}

func (p *PipelinesProvider) RemovePAC(ctx context.Context, f fn.Function, metadata any) error {
	p.RemovePACInvoked = true
	return p.RemovePACFn(f)
}
