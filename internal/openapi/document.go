package openapi

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document is the subset of a committed OpenAPI 3.1 document that
// gonext generate page reads.
type Document struct {
	Paths      map[string]PathItem `yaml:"paths"`
	Components struct {
		Schemas map[string]*Schema `yaml:"schemas"`
	} `yaml:"components"`
}

// PathItem holds the seven HTTP-method operations a path may declare.
type PathItem struct {
	Get     *Operation `yaml:"get"`
	Put     *Operation `yaml:"put"`
	Post    *Operation `yaml:"post"`
	Delete  *Operation `yaml:"delete"`
	Options *Operation `yaml:"options"`
	Head    *Operation `yaml:"head"`
	Patch   *Operation `yaml:"patch"`
}

// Operation is one HTTP operation. Method and Path are filled in by
// Document.Operation and are not part of the document itself.
type Operation struct {
	OperationID string              `yaml:"operationId"`
	Summary     string              `yaml:"summary"`
	Tags        []string            `yaml:"tags"`
	Parameters  []Parameter         `yaml:"parameters"`
	RequestBody *RequestBody        `yaml:"requestBody"`
	Responses   map[string]Response `yaml:"responses"`
	Method      string              `yaml:"-"`
	Path        string              `yaml:"-"`
}

// Parameter is one operation parameter.
type Parameter struct {
	Name     string  `yaml:"name"`
	In       string  `yaml:"in"`
	Required bool    `yaml:"required"`
	Schema   *Schema `yaml:"schema"`
}

// RequestBody is an operation's request body, keyed by media type.
type RequestBody struct {
	Required bool                 `yaml:"required"`
	Content  map[string]MediaType `yaml:"content"`
}

// Response is one status code's response, keyed by media type.
type Response struct {
	Content map[string]MediaType `yaml:"content"`
}

// MediaType holds the schema for one entry of a content map.
type MediaType struct {
	Schema *Schema `yaml:"schema"`
}

// Schema is the subset of an OpenAPI schema object gonext generate
// page reads. A $ref is resolved with Document.Resolve.
type Schema struct {
	Ref         string             `yaml:"$ref"`
	Type        string             `yaml:"-"`
	Nullable    bool               `yaml:"-"`
	Format      string             `yaml:"format"`
	Description string             `yaml:"description"`
	Properties  map[string]*Schema `yaml:"properties"`
	Required    []string           `yaml:"required"`
	ReadOnly    bool               `yaml:"readOnly"`
	Items       *Schema            `yaml:"items"`
}

// schemaAlias mirrors Schema without its custom UnmarshalYAML, so
// UnmarshalYAML can decode into it without recursing.
type schemaAlias Schema

// UnmarshalYAML decodes Schema.type as either a scalar or, per
// OpenAPI 3.1, a list such as [array, "null"], keeping the non-null
// entry as Type and setting Nullable.
func (s *Schema) UnmarshalYAML(node *yaml.Node) error {
	var raw struct {
		schemaAlias `yaml:",inline"`
		Type        yaml.Node `yaml:"type"`
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}
	*s = Schema(raw.schemaAlias)
	switch raw.Type.Kind {
	case yaml.ScalarNode:
		s.Type = raw.Type.Value
	case yaml.SequenceNode:
		for _, item := range raw.Type.Content {
			if item.Value == "null" {
				s.Nullable = true
				continue
			}
			s.Type = item.Value
		}
	}
	return nil
}

// Load reads and parses DocumentPath under root.
func Load(root string) (*Document, error) {
	data, err := os.ReadFile(filepath.Join(root, DocumentPath))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s not found; run `gonext openapi`", DocumentPath)
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", DocumentPath, err)
	}
	var doc Document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("reading %s: %w", DocumentPath, err)
	}
	return &doc, nil
}

// Operation returns the operation with the given operationId, with
// its Method (upper-case) and Path filled in from where it was found.
func (d *Document) Operation(id string) (*Operation, error) {
	paths := make([]string, 0, len(d.Paths))
	for p := range d.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		item := d.Paths[p]
		for _, m := range []struct {
			name string
			op   *Operation
		}{
			{"GET", item.Get},
			{"PUT", item.Put},
			{"POST", item.Post},
			{"DELETE", item.Delete},
			{"OPTIONS", item.Options},
			{"HEAD", item.Head},
			{"PATCH", item.Patch},
		} {
			if m.op == nil || m.op.OperationID != id {
				continue
			}
			op := *m.op
			op.Method = m.name
			op.Path = p
			return &op, nil
		}
	}
	return nil, fmt.Errorf("no operation %q in %s", id, DocumentPath)
}

// Resolve follows a $ref one level to its component schema. A nil
// schema or one with no $ref is returned unchanged; any ref form
// other than #/components/schemas/<Name> is an error.
func (d *Document) Resolve(s *Schema) (*Schema, error) {
	if s == nil || s.Ref == "" {
		return s, nil
	}
	name, ok := strings.CutPrefix(s.Ref, "#/components/schemas/")
	if ok {
		if resolved, ok := d.Components.Schemas[name]; ok {
			return resolved, nil
		}
	}
	return nil, fmt.Errorf("unsupported $ref %q in %s", s.Ref, DocumentPath)
}
