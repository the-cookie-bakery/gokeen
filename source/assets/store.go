package assets

import (
	"fmt"

	"gokeen/engine/animations"
)

type Store struct {
	pack *Pack
}

func NewStore(p *Pack) (*Store, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &Store{pack: p}, nil
}

func (s *Store) Pack() *Pack {
	if s == nil {
		return nil
	}
	return s.pack
}

func (s *Store) Mesh(id MeshID) (Mesh, error) {
	if s == nil || s.pack == nil {
		return Mesh{}, fmt.Errorf("mesh %q: %w", id, ErrNoStore)
	}
	m, ok := s.pack.Meshes[id]
	if !ok {
		return Mesh{}, fmt.Errorf("mesh %q: %w", id, ErrAssetNotFound)
	}
	return m, nil
}

func (s *Store) Material(id MaterialID) (Material, error) {
	if s == nil || s.pack == nil {
		return Material{}, fmt.Errorf("material %q: %w", id, ErrNoStore)
	}
	m, ok := s.pack.Materials[id]
	if !ok {
		return Material{}, fmt.Errorf("material %q: %w", id, ErrAssetNotFound)
	}
	return m, nil
}

func (s *Store) Texture(id TextureID) (Texture, error) {
	if s == nil || s.pack == nil {
		return Texture{}, fmt.Errorf("texture %q: %w", id, ErrNoStore)
	}
	t, ok := s.pack.Textures[id]
	if !ok {
		return Texture{}, fmt.Errorf("texture %q: %w", id, ErrAssetNotFound)
	}
	return t, nil
}

func (s *Store) Skeleton(id animations.SkeletonID) (animations.Skeleton, error) {
	if s == nil || s.pack == nil {
		return animations.Skeleton{}, fmt.Errorf("skeleton %q: %w", id, ErrNoStore)
	}
	v, ok := s.pack.Skeletons[id]
	if !ok {
		return animations.Skeleton{}, fmt.Errorf("skeleton %q: %w", id, ErrAssetNotFound)
	}
	return v, nil
}

func (s *Store) Clip(id animations.ClipID) (animations.Clip, error) {
	if s == nil || s.pack == nil {
		return animations.Clip{}, fmt.Errorf("clip %q: %w", id, ErrNoStore)
	}
	v, ok := s.pack.Clips[id]
	if !ok {
		return animations.Clip{}, fmt.Errorf("clip %q: %w", id, ErrAssetNotFound)
	}
	return v, nil
}

func (s *Store) Audio(id AudioClipID) (AudioClip, error) {
	if s == nil || s.pack == nil {
		return AudioClip{}, fmt.Errorf("audio clip %q: %w", id, ErrNoStore)
	}
	v, ok := s.pack.Audio[id]
	if !ok {
		return AudioClip{}, fmt.Errorf("audio clip %q: %w", id, ErrAssetNotFound)
	}
	return v, nil
}
