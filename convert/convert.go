package convert

import (
	"fmt"
	"reflect"

	"entgo.io/ent/entc/gen"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Converter struct {
	*descriptorpb.FileDescriptorProto
	usedNames map[string]struct{}

	messageMap map[reflect.Type]*descriptorpb.DescriptorProto
}

func New(fdp *descriptorpb.FileDescriptorProto) *Converter {
	c := &Converter{
		FileDescriptorProto: fdp,
		usedNames:           make(map[string]struct{}),
		messageMap:          make(map[reflect.Type]*descriptorpb.DescriptorProto),
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

func (c *Converter) getMessage(t reflect.Type) (*descriptorpb.DescriptorProto, bool) {
	if msg, ok := c.messageMap[t]; ok {
		return msg, true
	}
	return nil, false
}

func (c *Converter) addMessage(msg *descriptorpb.DescriptorProto, t reflect.Type) *descriptorpb.DescriptorProto {
	msg.Name = ptr(c.resolveMessageName(msg.GetName()))

	c.messageMap[t] = msg
	c.FileDescriptorProto.MessageType = append(c.FileDescriptorProto.MessageType, msg)
	return msg
}

var (
	snake  = gen.Funcs["snake"].(func(string) string)
	pascal = gen.Funcs["pascal"].(func(string) string)
	camel  = gen.Funcs["camel"].(func(string) string)
)
