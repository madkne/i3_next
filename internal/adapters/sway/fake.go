package sway

import "context"

type FakeAdapter struct {
	RunCommandFn    func(ctx context.Context, command string) (Message, error)
	GetWorkspacesFn func(ctx context.Context) ([]Workspace, error)
	GetOutputsFn    func(ctx context.Context) ([]Output, error)
	GetTreeFn       func(ctx context.Context) (*Node, error)
	GetInputsFn     func(ctx context.Context) ([]Input, error)
	GetVersionFn    func(ctx context.Context) (*Version, error)
	SubscribeFn     func(ctx context.Context, events []string) (<-chan Message, error)
	CloseFn         func() error
}

func (f *FakeAdapter) RunCommand(ctx context.Context, command string) (Message, error) {
	if f.RunCommandFn != nil {
		return f.RunCommandFn(ctx, command)
	}
	return Message{}, nil
}
func (f *FakeAdapter) GetWorkspaces(ctx context.Context) ([]Workspace, error) {
	if f.GetWorkspacesFn != nil {
		return f.GetWorkspacesFn(ctx)
	}
	return nil, nil
}
func (f *FakeAdapter) GetOutputs(ctx context.Context) ([]Output, error) {
	if f.GetOutputsFn != nil {
		return f.GetOutputsFn(ctx)
	}
	return nil, nil
}
func (f *FakeAdapter) GetTree(ctx context.Context) (*Node, error) {
	if f.GetTreeFn != nil {
		return f.GetTreeFn(ctx)
	}
	return nil, nil
}
func (f *FakeAdapter) GetInputs(ctx context.Context) ([]Input, error) {
	if f.GetInputsFn != nil {
		return f.GetInputsFn(ctx)
	}
	return nil, nil
}
func (f *FakeAdapter) GetVersion(ctx context.Context) (*Version, error) {
	if f.GetVersionFn != nil {
		return f.GetVersionFn(ctx)
	}
	return nil, nil
}
func (f *FakeAdapter) Subscribe(ctx context.Context, events []string) (<-chan Message, error) {
	if f.SubscribeFn != nil {
		return f.SubscribeFn(ctx, events)
	}
	return nil, nil
}
func (f *FakeAdapter) Close() error {
	if f.CloseFn != nil {
		return f.CloseFn()
	}
	return nil
}