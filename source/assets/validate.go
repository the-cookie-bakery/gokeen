package assets

import (
	"errors"
	"fmt"
	"math"

	"gokeen/engine/animations"
)

func (p *Pack) Validate() error {
	if p == nil {
		return errors.New("nil asset pack")
	}
	var errs []error
	for id, m := range p.Meshes {
		if id == "" {
			errs = append(errs, errors.New("mesh has empty ID"))
		}
		if m.ID != id {
			errs = append(errs, fmt.Errorf("mesh %q map key does not match ID %q", id, m.ID))
		}
		if !m.Bounds.IsValid() || !m.Bounds.Min.IsFinite() || !m.Bounds.Max.IsFinite() {
			errs = append(errs, fmt.Errorf("mesh %q has invalid bounds", id))
		}
		for i, v := range m.Vertices {
			if !v.Position.IsFinite() || !v.Normal.IsFinite() {
				errs = append(errs, fmt.Errorf("mesh %q vertex %d is not finite", id, i))
				break
			}
			for _, w := range v.BoneWeights {
				if math.IsNaN(float64(w)) || math.IsInf(float64(w), 0) {
					errs = append(errs, fmt.Errorf("mesh %q vertex %d has invalid bone weight", id, i))
					break
				}
			}
		}
		for i, idx := range m.Indices {
			if int(idx) >= len(m.Vertices) {
				errs = append(errs, fmt.Errorf("mesh %q index %d references vertex %d of %d", id, i, idx, len(m.Vertices)))
				break
			}
		}
		if len(m.Indices)%3 != 0 {
			errs = append(errs, fmt.Errorf("mesh %q index count %d is not divisible by 3", id, len(m.Indices)))
		}
		var covered uint64
		for _, sm := range m.Submeshes {
			end := uint64(sm.IndexStart) + uint64(sm.IndexCount)
			if end > uint64(len(m.Indices)) {
				errs = append(errs, fmt.Errorf("mesh %q submesh exceeds index buffer", id))
				break
			}
			if sm.Material != "" {
				if _, ok := p.Materials[sm.Material]; !ok {
					errs = append(errs, fmt.Errorf("mesh %q references missing material %q", id, sm.Material))
				}
			}
			covered += uint64(sm.IndexCount)
		}
		if len(m.Submeshes) > 0 && int(covered) > len(m.Indices) {
			errs = append(errs, fmt.Errorf("mesh %q submeshes overlap or exceed the index buffer", id))
		}
		if m.Skeleton != "" {
			skel, ok := p.Skeletons[m.Skeleton]
			if !ok {
				errs = append(errs, fmt.Errorf("mesh %q references missing skeleton %q", id, m.Skeleton))
			} else {
				for vi, v := range m.Vertices {
					var sum float32
					for bi, boneIndex := range v.BoneIndices {
						weight := v.BoneWeights[bi]
						if weight < 0 || weight > 1.0001 {
							errs = append(errs, fmt.Errorf("mesh %q vertex %d has invalid bone weight %v", id, vi, weight))
							break
						}
						if weight > 0 && int(boneIndex) >= len(skel.Bones) {
							errs = append(errs, fmt.Errorf("mesh %q vertex %d references bone %d of %d", id, vi, boneIndex, len(skel.Bones)))
							break
						}
						sum += weight
					}
					if sum > 1.001 {
						errs = append(errs, fmt.Errorf("mesh %q vertex %d bone weights sum to %v", id, vi, sum))
						break
					}
				}
			}
		}
	}
	for id, m := range p.Materials {
		if id == "" {
			errs = append(errs, errors.New("material has empty ID"))
		}
		if !m.AlphaMode.valid() {
			errs = append(errs, fmt.Errorf("material %q has invalid alpha mode", id))
		}
		for i := range m.BaseColor {
			if !finite32(m.BaseColor[i]) {
				errs = append(errs, fmt.Errorf("material %q base color is not finite", id))
				break
			}
		}
		for i, layer := range m.Layers {
			if !layer.Blend.valid() {
				errs = append(errs, fmt.Errorf("material %q layer %d has invalid blend mode", id, i))
			}
			if layer.UVSet > 1 {
				errs = append(errs, fmt.Errorf("material %q layer %d uses unsupported UV set %d", id, i, layer.UVSet))
			}
			if layer.Texture != "" {
				if _, ok := p.Textures[layer.Texture]; !ok {
					errs = append(errs, fmt.Errorf("material %q references missing texture %q", id, layer.Texture))
				}
			}
		}
	}
	for id, t := range p.Textures {
		if id == "" {
			errs = append(errs, errors.New("texture has empty ID"))
		}
		if t.Width == 0 || t.Height == 0 {
			errs = append(errs, fmt.Errorf("texture %q has zero dimensions", id))
		}
		if t.Format != TextureFormatPNG && t.Format != TextureFormatRGBA8 {
			errs = append(errs, fmt.Errorf("texture %q has unsupported format %d", id, t.Format))
		}
		if t.Format == TextureFormatRGBA8 && len(t.Data) != int(t.Width*t.Height*4) {
			errs = append(errs, fmt.Errorf("texture %q RGBA data size mismatch", id))
		}
	}
	for id, s := range p.Skeletons {
		if len(s.Bones) == 0 {
			errs = append(errs, fmt.Errorf("skeleton %q has no bones", id))
		}
		for i, b := range s.Bones {
			if b.Parent < -1 || b.Parent >= int32(i) {
				errs = append(errs, fmt.Errorf("skeleton %q bone %d has invalid parent %d", id, i, b.Parent))
			}
			if !b.Local.IsFinite() || !b.InverseBind.IsFinite() {
				errs = append(errs, fmt.Errorf("skeleton %q bone %d is not finite", id, i))
			}
		}
	}
	for id, c := range p.Clips {
		if c.Duration < 0 || !finite32(c.Duration) {
			errs = append(errs, fmt.Errorf("clip %q has invalid duration", id))
		}
		for ti, track := range c.Tracks {
			if track.Bone < 0 {
				errs = append(errs, fmt.Errorf("clip %q track %d has negative bone index", id, ti))
				continue
			}
			if err := validateVectorKeys(track.Position); err != nil {
				errs = append(errs, fmt.Errorf("clip %q track %d position: %w", id, ti, err))
			}
			if err := validateVectorKeys(track.Scale); err != nil {
				errs = append(errs, fmt.Errorf("clip %q track %d scale: %w", id, ti, err))
			}
			if err := validateQuatKeys(track.Rotation); err != nil {
				errs = append(errs, fmt.Errorf("clip %q track %d rotation: %w", id, ti, err))
			}
		}
	}
	_ = p.Clips
	_ = p.Audio
	return errors.Join(errs...)
}

func validateVectorKeys(keys []animations.VectorKey) error {
	last := float32(-1)
	for i, k := range keys {
		if !finite32(k.Time) || k.Time < last {
			return fmt.Errorf("invalid or unsorted key time at %d", i)
		}
		if !k.Value.IsFinite() {
			return fmt.Errorf("non-finite value at key %d", i)
		}
		last = k.Time
	}
	return nil
}

func validateQuatKeys(keys []animations.QuatKey) error {
	last := float32(-1)
	for i, k := range keys {
		if !finite32(k.Time) || k.Time < last {
			return fmt.Errorf("invalid or unsorted key time at %d", i)
		}
		if !k.Value.IsFinite() {
			return fmt.Errorf("non-finite value at key %d", i)
		}
		last = k.Time
	}
	return nil
}

func finite32(v float32) bool {
	f := float64(v)
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
