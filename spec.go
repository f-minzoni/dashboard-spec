package main

import (
	"encoding/json"
	"log"
	"sort"
)

// OpenAPI 3.0 spec document.
type Spec struct {
	Version string `json:"openapi"`
	Info    struct {
		Title   string
		Version string
	}
	Components struct {
		Schemas map[string]Schema
	}
}

// OpenAPI 3.0 schema.
type Schema struct {
	Title       string
	Default     interface{}
	Description string
	Items       *Schema
	Properties  map[string]*Schema
	ReadOnly    bool
	Required    []string
	Type        string
}

// Used for the purpose of flattening the properties of a schema. The location
// field makes it possible to reconstruct later. This facilitates generating
// setter/appender methods for deeply nested properties.
type MappedSchema struct {
	Name     string
	Location []string
	Schema   *Schema
}

// Return a schema's default value as JSON.
func (s Schema) DefaultJSON() string {
	b, err := json.Marshal(s.Default)
	if err != nil {
		log.Fatalln(err)
	}
	return string(b)
}

// If title is set, it's assumed it carries more meaning than the property name
// itself. And therefore more suitable for humans. This is useful for naming
// arguments and functions.
func (s Schema) HumanName(name string) string {
	if s.Title != "" {
		return s.Title
	} else {
		return name
	}
}

// Returns all top-level properties except objects and arrays of objects. These
// are intended to be used as arguments for the schema object's constructor.
func (s Schema) TopLevelSimpleProperties() map[string]*Schema {
	p := map[string]*Schema{}
	for n, s := range s.Properties {
		if !s.ReadOnly && s.Type != "object" &&
			(s.Type != "array" || s.Type == "array" && s.Items.Type != "object") {
			p[n] = s
		}
	}
	return p
}

// Returns all properties that are readOnly and have a default property. It's
// intended that these are set, but not explicitly configurable. For example, a
// panel's "type" field.
func (s Schema) ReadOnlyWithDefaultProperties() []MappedSchema {
	return flatten(&s, func(s *Schema) bool {
		return s.ReadOnly && s.Default != nil
	})
}

// Returns all top-level object properties. It's anticipated that these have
// setter methods nested inside their parent schema object.
func (s Schema) TopLevelObjectProperties() map[string]*Schema {
	p := map[string]*Schema{}
	for n, s := range s.Properties {
		if !s.ReadOnly && s.Type == "object" {
			p[n] = s
		}
	}
	return p
}

// Returns all nested properties except arrays of objects. It's anticipated
// that the parent schema object is a top-level object property and that the
// properties returned here will be arguments in the parent's setter method.
func (s Schema) NestedSimpleProperties() []MappedSchema {
	return flatten(&s, func(s *Schema) bool {
		return !s.ReadOnly && s.Type != "object" &&
			(s.Type != "array" || s.Type == "array" && s.Items.Type != "object")
	})
}

// Returns nested properties that are arrays of objects. It's anticipated that
// these are used to create appender methods for constructing those objects and
// appending them.
func (s Schema) NestedComplexArrayProperties() []MappedSchema {
	return flatten(&s, func(s *Schema) bool {
		return !s.ReadOnly && s.Type == "array" && s.Items.Type == "object"
	})
}

// Recursively flattens nested properties.
func flatten(s *Schema, filter func(*Schema) bool) (ms []MappedSchema) {
	var flatten func(*Schema, []string)
	flatten = func(s *Schema, locationPrefix []string) {
		for n, s := range s.Properties {
			if filter(s) {
				ms = append(ms, MappedSchema{
					Name:     n,
					Location: append(locationPrefix, n),
					Schema:   s,
				})
			} else if s.Type == "object" {
				flatten(s, append(locationPrefix, n))
			}
		}
	}
	flatten(s, []string{})
	sort.SliceStable(ms, func(i, j int) bool { return ms[i].Name < ms[j].Name })
	return ms
}
