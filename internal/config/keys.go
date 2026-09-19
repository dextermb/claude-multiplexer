package config

import (
	"reflect"
	"strings"
)

// KeyPath is one settings key, in the dot notation set_config takes, with its
// JSON type. A map key is a "<key>" placeholder, so a session substitutes its
// own name. See docs/mcp/tools/settings.md.
type KeyPath struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// MapKeyPlaceholder stands for a key the user names, such as a block bucket or a
// layout name.
const MapKeyPlaceholder = "<key>"

// Keys lists the settings key paths set_config and unset_config accept. It walks
// the Config type, so the list mirrors the schema those tools validate against.
// See docs/mcp/tools/settings.md.
func Keys() []KeyPath {
	var keys []KeyPath
	walkKeys(reflect.TypeOf(Config{}), "", &keys)
	return keys
}

func walkKeys(t reflect.Type, prefix string, keys *[]KeyPath) {
	switch t.Kind() {
	case reflect.Pointer:
		walkKeys(t.Elem(), prefix, keys)
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name := jsonName(field)
			if name == "" {
				continue
			}
			walkKeys(field.Type, join(prefix, name), keys)
		}
	case reflect.Map:
		walkKeys(t.Elem(), join(prefix, MapKeyPlaceholder), keys)
	default:
		*keys = append(*keys, KeyPath{Path: prefix, Type: jsonType(t)})
	}
}

func jsonName(field reflect.StructField) string {
	if field.PkgPath != "" {
		return ""
	}
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return field.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "-" {
		return ""
	}
	if name == "" {
		return field.Name
	}
	return name
}

func jsonType(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	default:
		return "object"
	}
}

func join(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
