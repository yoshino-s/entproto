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

type ActionAfter string

const (
	ActionAfterGet    ActionAfter = "after_get"
	ActionAfterDelete ActionAfter = "after_delete"
	ActionAfterCreate ActionAfter = "after_create"
	ActionAfterUpdate ActionAfter = "after_update"
	ActionAfterList   ActionAfter = "after_list"
)

type HookAfter interface {
	HookAfter(ctx context.Context, action ActionAfter, request any, response any) error
}

type HookAfterFunc func(ctx context.Context, action ActionAfter, request any, response any) error

func (f HookAfterFunc) HookAfter(ctx context.Context, action ActionAfter, request any, response any) error {
	return f(ctx, action, request, response)
}
