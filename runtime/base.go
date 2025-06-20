package runtime

import (
	"context"
)

type BaseService struct {
	hooks      []Hook
	afterHooks []HookAfter
}

func NewBaseService() *BaseService {
	return &BaseService{
		hooks:      []Hook{},
		afterHooks: []HookAfter{},
	}
}

func (svc *BaseService) AddHook(hook Hook) {
	svc.hooks = append(svc.hooks, hook)
}

func (svc *BaseService) AddAfterHook(hook HookAfter) {
	svc.afterHooks = append(svc.afterHooks, hook)
}

func (svc *BaseService) RunHooks(ctx context.Context, action Action, request any, query any) error {
	for _, hook := range svc.hooks {
		if err := hook.Hook(ctx, action, request, query); err != nil {
			return err
		}
	}
	return nil
}

func (svc *BaseService) RunHooksAfter(ctx context.Context, action ActionAfter, request any, response any) error {
	for _, hook := range svc.afterHooks {
		if err := hook.HookAfter(ctx, action, request, response); err != nil {
			return err
		}
	}
	return nil
}
