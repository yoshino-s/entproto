package struct_converter

import (
	"fmt"
	"reflect"
	"testing"
)

func TestMarshalGoType(t *testing.T) {
	gt := reflect.TypeOf(TestStruct{})
	marshaled, err := MarshalGoType(gt)
	if err != nil {
		t.Fatalf("failed marshaling go type: %v", err)
	}

	fmt.Printf("Marshaled Go Type: %+v\n", marshaled)
}
