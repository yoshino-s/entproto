package entproto

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"github.com/go-viper/mapstructure/v2"
	"github.com/gookit/goutil/arrutil"
)

const ExtraFilterAnnotation = "ProtoExtraFilter"

func ExtraFilter(extraFields ...ent.Field) *extraFilter {
	return &extraFilter{
		ExtraFields: arrutil.Map(extraFields, func(f ent.Field) (*field.Descriptor, bool) {
			return f.Descriptor(), true
		}),
	}
}

type extraFilter struct {
	ExtraFields []*field.Descriptor `json:"extra_fields"`
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
