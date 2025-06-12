package runtime

import (
	"connectrpc.com/connect"
)

func WrapResult[T any](entity *T, err error) (*connect.Response[T], error) {
	if err != nil {
		if connectErr, ok := err.(*connect.Error); ok {
			return nil, connectErr
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(entity), nil
}
