package assets

import "errors"

var (
	ErrAssetNotFound = errors.New("asset not found")
	ErrNoStore       = errors.New("asset store is nil")
)
