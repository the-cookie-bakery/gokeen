package levels

import (
	"gokeen/assets"
	"gokeen/engine/animations"
)

type AssetResolver interface {
	HasMesh(id string) bool
	HasMaterial(id string) bool
	HasTexture(id string) bool
	HasSkeleton(id string) bool
	HasClip(id string) bool
}

type StoreResolver struct {
	Store *assets.Store
}

func (r StoreResolver) HasMesh(id string) bool {
	if r.Store == nil || r.Store.Pack() == nil {
		return false
	}
	_, ok := r.Store.Pack().Meshes[assets.MeshID(id)]
	return ok
}

func (r StoreResolver) HasMaterial(id string) bool {
	if r.Store == nil || r.Store.Pack() == nil {
		return false
	}
	_, ok := r.Store.Pack().Materials[assets.MaterialID(id)]
	return ok
}

func (r StoreResolver) HasTexture(id string) bool {
	if r.Store == nil || r.Store.Pack() == nil {
		return false
	}
	_, ok := r.Store.Pack().Textures[assets.TextureID(id)]
	return ok
}

func (r StoreResolver) HasSkeleton(id string) bool {
	if r.Store == nil || r.Store.Pack() == nil {
		return false
	}
	_, ok := r.Store.Pack().Skeletons[animations.SkeletonID(id)]
	return ok
}

func (r StoreResolver) HasClip(id string) bool {
	if r.Store == nil || r.Store.Pack() == nil {
		return false
	}
	_, ok := r.Store.Pack().Clips[animations.ClipID(id)]
	return ok
}
