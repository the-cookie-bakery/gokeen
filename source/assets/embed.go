package assets

import _ "embed"

//go:embed pack/gokeen.gkp
var embeddedPack []byte

func EmbeddedPack() []byte {
	return embeddedPack
}

func LoadEmbedded() (*Pack, error) {
	return ReadPack(embeddedPack)
}
