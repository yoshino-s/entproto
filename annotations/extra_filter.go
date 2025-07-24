package annotations

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/entc/gen"
	"github.com/go-viper/mapstructure/v2"
	"github.com/gookit/goutil/arrutil"
)

const ExtraFilterAnnotation = "ProtoExtraFilter"

func ExtraFilter(extraFields ...ent.Field) *extraFilter {
	return &extraFilter{
		ExtraFields: arrutil.Map(extraFields, func(f ent.Field) (*gen.Field, bool) {
			fd := f.Descriptor()
			annotations := map[string]any{}
			if fd.Annotations != nil {
				for _, v := range fd.Annotations {
					annotations[v.Name()] = v
				}
			}
			if _, ok := annotations[FilterAnnotation]; !ok {
				annotations[FilterAnnotation] = Filter()
			}
			enums := arrutil.Map(fd.Enums, func(e struct {
				N string
				V string
			}) (gen.Enum, bool) {
				return gen.Enum{
					Name:  e.N,
					Value: e.V,
				}, true
			})

			gf := &gen.Field{
				Name:          fd.Name,
				Annotations:   annotations,
				Type:          fd.Info,
				Unique:        fd.Unique,
				Optional:      fd.Optional,
				Nillable:      fd.Nillable,
				Default:       fd.Default != nil,
				Enums:         enums,
				UpdateDefault: fd.UpdateDefault != nil,
				Immutable:     fd.Immutable,
			}
			return gf, true
		}),
	}
}

type extraFilter struct {
	ExtraFields []*gen.Field `json:"extra_fields" mapstructure:"extra_fields"`
}

func (f *extraFilter) Name() string {
	return ExtraFilterAnnotation
}

func ExtractExtraFilterAnnotation(sch *gen.Type) (*extraFilter, error) {
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
