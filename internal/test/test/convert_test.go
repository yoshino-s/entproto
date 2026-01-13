package test

import (
	"testing"

	"github.com/yoshino-s/entproto/internal/test/ent"
	"github.com/yoshino-s/entproto/internal/test/ent/schema"
	"github.com/yoshino-s/entproto/internal/test/proto/entpb/entpbservice"

	"github.com/stretchr/testify/assert"
)

func TestConvertGroup(t *testing.T) {
	group := &ent.Group{
		ID:   1,
		Name: "test",
		Metadata: map[string]string{
			"key":  "value",
			"key2": "value2",
		},
		Tags: []string{"tag1", "tag2"},
		SomeStruct: schema.TestStruct{
			StringField: "test",
			IntField:    1,
			BoolField:   true,
			ListField:   []string{"item1", "item2"},
			MapField:    map[string]int32{"key1": 1, "key2": 2},
			NestedField: &schema.NestedStruct{
				A: "test",
			},
			ListNestedField: []schema.NestedStruct{
				{A: "test"},
				{A: "test2"},
			},
			MapNestedField: map[string]schema.NestedStruct{
				"key1": {A: "test"},
				"key2": {A: "test2"},
			},
			AnyField: "test",
		},
		MetadataStruct: &schema.GroupMetadata{
			Version: "1.0.0",
		},
	}

	protoGroup, err := entpbservice.ToProtoGroup(group)
	assert.NoError(t, err)

	assert.Equal(t, protoGroup.Id, int32(group.ID))
	assert.Equal(t, protoGroup.Name, group.Name)
	assert.Equal(t, protoGroup.Metadata, group.Metadata)
	assert.Equal(t, protoGroup.Tags, group.Tags)
	assert.Equal(t, protoGroup.SomeStruct, group.SomeStruct)
	assert.Equal(t, protoGroup.MetadataStruct, group.MetadataStruct)
}
