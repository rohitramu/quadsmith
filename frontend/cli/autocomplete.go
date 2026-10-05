package main

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

func getMessageFields(msg protoreflect.MessageDescriptor, prefix string) []string {
	var completions []string
	fields := msg.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		if field.Kind() == protoreflect.MessageKind && !field.IsList() {
			completions = append(completions, getMessageFields(field.Message(), path)...)
		} else {
			completions = append(completions, path)
		}
	}
	return completions
}
