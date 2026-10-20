package assets

type MeshID string
type MaterialID string
type TextureID string
type AudioClipID string

type TextureFormat uint8

const (
	TextureFormatPNG   TextureFormat = 1
	TextureFormatRGBA8 TextureFormat = 2
)

type AlphaMode uint8

const (
	AlphaOpaque AlphaMode = iota
	AlphaMask
	AlphaBlend
)

type BlendMode uint8

const (
	BlendReplace BlendMode = iota
	BlendAlpha
	BlendMultiply
	BlendAdd
	BlendMultiply2
)

func (m AlphaMode) valid() bool {
	return m <= AlphaBlend
}

func (m BlendMode) valid() bool {
	return m <= BlendMultiply2
}
