package annotations

import (
	"fmt"

	"entgo.io/ent/entc/gen"
	"github.com/go-viper/mapstructure/v2"
)

const FilterAnnotation = "ProtoFilter"

type FilterOption func(f *filter)

func Filter(opts ...FilterOption) *filter {
	f := &filter{
		Mode: FilterModeEQ,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func FilterContains() FilterOption {
	return func(f *filter) {
		f.Mode = FilterModeContains
	}
}

func WithFilterMode(mode FilterMode) FilterOption {
	return func(f *filter) {
		f.Mode = mode
	}
}

type FilterMode int

const (
	FilterModeEQ FilterMode = 1 << iota
	FilterModeContains
	FilterModeIn
)

type filter struct {
	Mode FilterMode
}

func (f *filter) Name() string {
	return FilterAnnotation
}

func ExtractFilterAnnotation(sch *gen.Field) (*filter, error) {
	annot, ok := sch.Annotations[FilterAnnotation]
	if !ok {
		return nil, nil // No filter annotation present
	}

	var out filter
	err := mapstructure.Decode(annot, &out)
	if err != nil {
		return nil, fmt.Errorf("entproto: unable to decode entproto.Filter annotation for schema %q: %w",
			sch.Name, err)
	}

	return &out, nil
}
