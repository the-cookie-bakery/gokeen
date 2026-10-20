package levels

import (
	"encoding/json"
	"fmt"
)

type EntityDefinition struct {
	Type        string
	DisplayName string
	Category    string
	Decode      func(json.RawMessage) (any, error)
}

type Registry struct {
	defs map[string]EntityDefinition
}

func NewRegistry() *Registry {
	return &Registry{defs: make(map[string]EntityDefinition)}
}

func (r *Registry) Register(def EntityDefinition) error {
	if def.Type == "" {
		return fmt.Errorf("entity definition has empty type")
	}
	if _, ok := r.defs[def.Type]; ok {
		return fmt.Errorf("entity type %q already registered", def.Type)
	}
	if def.Decode == nil {
		def.Decode = func(json.RawMessage) (any, error) { return struct{}{}, nil }
	}
	if def.DisplayName == "" {
		def.DisplayName = def.Type
	}
	r.defs[def.Type] = def
	return nil
}

func (r *Registry) Has(typeName string) bool {
	if r == nil {
		return false
	}
	_, ok := r.defs[typeName]
	return ok
}

func (r *Registry) Definition(typeName string) (EntityDefinition, bool) {
	if r == nil {
		return EntityDefinition{}, false
	}
	def, ok := r.defs[typeName]
	return def, ok
}

func (r *Registry) Decode(typeName string, raw json.RawMessage) (any, error) {
	def, ok := r.Definition(typeName)
	if !ok {
		return nil, fmt.Errorf("unknown entity type %q", typeName)
	}
	return def.Decode(raw)
}

func (r *Registry) Types() []EntityDefinition {
	if r == nil {
		return nil
	}
	out := make([]EntityDefinition, 0, len(r.defs))
	for _, def := range r.defs {
		out = append(out, def)
	}
	return out
}
