package runtime

import (
	"strings"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/entc/gen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	snake = gen.Funcs["snake"].(func(string) string)
)

func BuildFilters[T ~func(*sql.Selector)](validColumn func(string) bool, message proto.Message) []T {
	filters := []T{}
	msgReflect := message.ProtoReflect()
	msgReflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		name := snake(string(fd.Name()))

		if strings.HasSuffix(name, "_contains") {
			name = strings.TrimSuffix(name, "_contains")
			if validColumn(name) {
				filters = append(filters, T(sql.FieldContains(name, v.Interface().(string))))
			}
		} else {
			if validColumn(name) {
				filters = append(filters, T(sql.FieldEQ(name, v.Interface())))
			}
		}

		return true
	})
	return filters
}
