package runtime

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/yoshino-s/entproto/structpb_wrapper"
	"google.golang.org/protobuf/types/known/structpb"
)

func ToStructPbValue(v any) (*structpb.Value, error) {
	return structpb_wrapper.NewValue(v)
}

func FromStructPbValue(fro *structpb.Value, dst any) error {
	return mapstructure.Decode(fro.AsInterface(), dst)
}
