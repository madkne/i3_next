package sway

import "context"

type Adapter interface {
	RunCommand(ctx context.Context, command string) (Message, error)
	GetWorkspaces(ctx context.Context) ([]Workspace, error)
	GetOutputs(ctx context.Context) ([]Output, error)
	GetTree(ctx context.Context) (*Node, error)
	GetInputs(ctx context.Context) ([]Input, error)
	GetVersion(ctx context.Context) (*Version, error)
	Subscribe(ctx context.Context, events []string) (<-chan Message, error)
	Close() error
}