package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// encodeFull builds a yaml.Node tree for v that includes every field, ignoring
// the `omitempty` tag option. It is used when generating the default config so
// the file is written with all keys present at their zero values (false, "",
// null, ...). Normal saves still go through the struct's tags and keep
// omitempty behavior.
func encodeFull(v reflect.Value) (*yaml.Node, error) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
		}
		return encodeFull(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
		}
		return encodeFull(v.Elem())
	case reflect.Struct:
		node := &yaml.Node{Kind: yaml.MappingNode}
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" { // unexported
				continue
			}
			name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
			if name == "-" {
				continue
			}
			if name == "" {
				name = strings.ToLower(field.Name)
			}
			valNode, err := encodeFull(v.Field(i))
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: name},
				valNode,
			)
		}
		return node, nil
	case reflect.Map:
		node := &yaml.Node{Kind: yaml.MappingNode}
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool {
			return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
		})
		for _, k := range keys {
			keyNode := &yaml.Node{}
			if err := keyNode.Encode(k.Interface()); err != nil {
				return nil, err
			}
			valNode, err := encodeFull(v.MapIndex(k))
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, keyNode, valNode)
		}
		return node, nil
	case reflect.Slice, reflect.Array:
		node := &yaml.Node{Kind: yaml.SequenceNode}
		for i := 0; i < v.Len(); i++ {
			elemNode, err := encodeFull(v.Index(i))
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, elemNode)
		}
		return node, nil
	default:
		node := &yaml.Node{}
		if err := node.Encode(v.Interface()); err != nil {
			return nil, err
		}
		return node, nil
	}
}
