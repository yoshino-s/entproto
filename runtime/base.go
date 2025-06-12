package runtime

import (
	"context"
)

type BaseService struct {
	Hooks []Hook
}

func NewBaseService() *BaseService {
	return &BaseService{
		Hooks: []Hook{},
	}
}

func (svc *BaseService) AddHook(hook Hook) {
	svc.Hooks = append(svc.Hooks, hook)
}

func (svc *BaseService) RunHooks(ctx context.Context, action Action, request any, query any) error {
	for _, hook := range svc.Hooks {
		if err := hook.Hook(ctx, action, request, query); err != nil {
			return err
		}
	}
	return nil
}
