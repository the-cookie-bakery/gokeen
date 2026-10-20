package levels

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

func Load(data []byte, resolver AssetResolver, registry *Registry) (*Level, error) {
	var level Level
	if err := json.Unmarshal(data, &level); err != nil {
		return nil, fmt.Errorf("decode level: %w", err)
	}
	if err := level.Validate(resolver, registry); err != nil {
		return nil, err
	}
	return &level, nil
}

func (l *Level) Validate(resolver AssetResolver, registry *Registry) error {
	if l == nil {
		return errors.New("nil level")
	}
	var errs []error
	if l.FormatVersion != FormatVersion {
		errs = append(errs, fmt.Errorf("unsupported level format version %d (want %d)", l.FormatVersion, FormatVersion))
	}
	if l.ID == "" {
		errs = append(errs, errors.New("level id is empty"))
	}
	if l.Name == "" {
		errs = append(errs, errors.New("level name is empty"))
	}
	if !l.Spawn.Position.IsFinite() || !l.Spawn.Rotation.IsFinite() {
		errs = append(errs, errors.New("spawn transform is not finite"))
	}
	if !l.Environment.Ambient.IsFinite() {
		errs = append(errs, errors.New("ambient color is not finite"))
	}
	if l.Environment.Sun != nil {
		if !l.Environment.Sun.Direction.IsFinite() || l.Environment.Sun.Direction.Vector().LengthSquared() < 1e-8 {
			errs = append(errs, errors.New("sun direction is invalid"))
		}
		if !l.Environment.Sun.Color.IsFinite() || l.Environment.Sun.Intensity < 0 {
			errs = append(errs, errors.New("sun color/intensity is invalid"))
		}
	}
	if l.Environment.Fog != nil {
		if !l.Environment.Fog.Color.IsFinite() || l.Environment.Fog.Start < 0 || l.Environment.Fog.End <= l.Environment.Fog.Start {
			errs = append(errs, errors.New("fog range is invalid"))
		}
	}

	seen := make(map[string]string)
	addID := func(kind, id string) {
		if id == "" {
			errs = append(errs, fmt.Errorf("%s has empty id", kind))
			return
		}
		if prev, ok := seen[id]; ok {
			errs = append(errs, fmt.Errorf("duplicate id %q used by %s and %s", id, prev, kind))
			return
		}
		seen[id] = kind
	}

	for i, g := range l.Geometry {
		addID("geometry", g.ID)
		if !g.Transform.IsFinite() {
			errs = append(errs, fmt.Errorf("geometry %q transform is not finite", g.ID))
			continue
		}
		if g.Mesh == "" {
			errs = append(errs, fmt.Errorf("geometry %q has no mesh", g.ID))
		} else if resolver != nil && !resolver.HasMesh(g.Mesh) {
			errs = append(errs, fmt.Errorf("geometry %q references missing mesh %q", g.ID, g.Mesh))
		}
		if g.Collision != "" && resolver != nil && !resolver.HasMesh(g.Collision) {
			errs = append(errs, fmt.Errorf("geometry %q references missing collision mesh %q", g.ID, g.Collision))
		}
		_ = i
	}

	for _, e := range l.Entities {
		addID("entity", e.ID)
		if e.Type == "" {
			errs = append(errs, fmt.Errorf("entity %q has no type", e.ID))
		} else if registry != nil {
			if !registry.Has(e.Type) {
				errs = append(errs, fmt.Errorf("entity %q has unknown type %q", e.ID, e.Type))
			} else if _, err := registry.Decode(e.Type, e.Properties); err != nil {
				errs = append(errs, fmt.Errorf("entity %q properties: %w", e.ID, err))
			}
		}
		if !e.Transform.IsFinite() {
			errs = append(errs, fmt.Errorf("entity %q transform is not finite", e.ID))
		}
	}

	for _, light := range l.Lights {
		addID("light", light.ID)
		if !light.Transform.IsFinite() || !light.Color.IsFinite() {
			errs = append(errs, fmt.Errorf("light %q has non-finite values", light.ID))
			continue
		}
		switch light.Type {
		case LightDirectional:
			// Direction is derived from the transform rotation.
		case LightPoint, LightSpot:
			if light.Range <= 0 || math.IsNaN(float64(light.Range)) || math.IsInf(float64(light.Range), 0) {
				errs = append(errs, fmt.Errorf("light %q has invalid range", light.ID))
			}
			if light.Type == LightSpot {
				if light.InnerCone < 0 || light.OuterCone <= light.InnerCone || light.OuterCone > 180 {
					errs = append(errs, fmt.Errorf("light %q has invalid cone angles", light.ID))
				}
			}
		default:
			errs = append(errs, fmt.Errorf("light %q has unknown type %q", light.ID, light.Type))
		}
		if light.Intensity < 0 {
			errs = append(errs, fmt.Errorf("light %q has negative intensity", light.ID))
		}
	}

	for _, t := range l.Triggers {
		addID("trigger", t.ID)
		if t.Type == "" {
			errs = append(errs, fmt.Errorf("trigger %q has no type", t.ID))
		}
		if !t.Transform.IsFinite() {
			errs = append(errs, fmt.Errorf("trigger %q transform is not finite", t.ID))
		}
		if t.Size == nil && t.Radius <= 0 {
			errs = append(errs, fmt.Errorf("trigger %q has neither size nor positive radius", t.ID))
		}
		if t.Size != nil && (!t.Size.IsFinite() || t.Size.Vector().X < 0 || t.Size.Vector().Y < 0 || t.Size.Vector().Z < 0) {
			errs = append(errs, fmt.Errorf("trigger %q size is invalid", t.ID))
		}
	}
	return errors.Join(errs...)
}
