package runtime

import (
	"github.com/go-viper/mapstructure/v2"
	gojson "github.com/goccy/go-json"
	"github.com/yoshino-s/entproto/structpb_wrapper"
	"google.golang.org/protobuf/types/known/structpb"
)

func copy(in, out any) error {
	b, err := gojson.Marshal(in)
	if err != nil {
		return err
	}
	return gojson.Unmarshal(b, out)
}

func ToStructPbValue(v any) (*structpb.Value, error) {
	mapValue := map[string]any{}
	if err := copy(v, &mapValue); err != nil {
		return nil, err
	}
	return structpb_wrapper.NewValue(mapValue)
}

func FromStructPbValue(fro *structpb.Value, dst any) error {
	v := fro.AsInterface()
	if err := copy(v, &dst); err != nil {
		return err
	}
	return mapstructure.Decode(v, dst)
}
