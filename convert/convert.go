package convert

import (
	"fmt"

	"entgo.io/ent/entc/gen"
	"github.com/yoshino-s/entproto/convert/struct_converter"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Converter struct {
	*descriptorpb.FileDescriptorProto
	usedNames map[string]struct{}

	messageMap map[string]*descriptorpb.DescriptorProto
}

func New(fdp *descriptorpb.FileDescriptorProto) *Converter {
	c := &Converter{
		FileDescriptorProto: fdp,
		usedNames:           make(map[string]struct{}),
		messageMap:          make(map[string]*descriptorpb.DescriptorProto),
	}

	c.FileDescriptorProto.Dependency = append(c.FileDescriptorProto.Dependency, "google/protobuf/wrappers.proto")
	c.FileDescriptorProto.Dependency = append(c.FileDescriptorProto.Dependency, "google/protobuf/struct.proto")

	return c
}

func (c *Converter) resolveMessageName(name string) string {
	if _, ok := c.usedNames[name]; !ok {
		c.usedNames[name] = struct{}{}
		return name
	}

	cnt := 0
	for {
		newName := fmt.Sprintf("%s%d", name, cnt)

		if _, ok := c.usedNames[newName]; !ok {
			c.usedNames[newName] = struct{}{}
			return newName
		}
		cnt++
	}
}

func (c *Converter) GetMessageMap() map[string]*descriptorpb.DescriptorProto {
	return c.messageMap
}

func (c *Converter) AddMessage(typ *struct_converter.MarshaledGoType, msg *descriptorpb.DescriptorProto) (bool, error) {
	if _, exists := c.messageMap[typ.GoTypeString()]; exists {
		return false, nil
	}
	msg.Name = ptr(c.resolveMessageName(msg.GetName()))

	c.messageMap[typ.GoTypeString()] = msg
	c.FileDescriptorProto.MessageType = append(c.FileDescriptorProto.MessageType, msg)

	return true, nil
}

var (
	snake  = gen.Funcs["snake"].(func(string) string)
	pascal = gen.Funcs["pascal"].(func(string) string)
	camel  = gen.Funcs["camel"].(func(string) string)
)
