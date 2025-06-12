package entproto

import (
	"fmt"

	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"github.com/go-viper/mapstructure/v2"
)

const ExtraFilterAnnotation = "ProtoExtraFilter"

func ExtraFilter(extraFields map[string]field.Type) *extraFilter {
	return &extraFilter{
		ExtraFields: extraFields,
	}
}

type extraFilter struct {
	ExtraFields map[string]field.Type
}

func (f *extraFilter) Name() string {
	return ExtraFilterAnnotation
}

func extractExtraFilterAnnotation(sch *gen.Type) (*extraFilter, error) {
	annot, ok := sch.Annotations[ExtraFilterAnnotation]
	if !ok {
		return nil, nil // No filter annotation present
	}

	var out extraFilter
	err := mapstructure.Decode(annot, &out)
	if err != nil {
		return nil, fmt.Errorf("entproto: unable to decode entproto.Filter annotation for schema %q: %w",
			sch.Name, err)
	}

	return &out, nil
}
