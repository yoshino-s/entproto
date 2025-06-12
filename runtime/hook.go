package runtime

import "context"

type Action string

const (
	ActionGet       Action = "get"
	ActionDelete    Action = "delete"
	ActionCreate    Action = "create"
	ActionUpdate    Action = "update"
	ActionList      Action = "list"
	ActionListCount Action = "list_count"
)

type Hook interface {
	Hook(ctx context.Context, action Action, request any, query any) error
}

type HookFunc func(ctx context.Context, action Action, request any, query any) error

func (f HookFunc) Hook(ctx context.Context, action Action, request any, query any) error {
	return f(ctx, action, request, query)
}
