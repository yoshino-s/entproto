package convert

import "errors"

var (
	ErrSchemaSkipped = errors.New("entproto: schema not annotated with Generate=true")
)
