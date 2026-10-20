package levels

import (
	"embed"
	"fmt"
)

//go:embed data/*.json
var embeddedLevels embed.FS

func EmbeddedLevelData(id LevelID) ([]byte, error) {
	data, err := embeddedLevels.ReadFile("data/" + string(id) + ".json")
	if err != nil {
		return nil, fmt.Errorf("embedded level %q: %w", id, err)
	}
	return data, nil
}

func LoadEmbedded(id LevelID, resolver AssetResolver, registry *Registry) (*Level, error) {
	data, err := EmbeddedLevelData(id)
	if err != nil {
		return nil, err
	}
	return Load(data, resolver, registry)
}

func EmbeddedLevelFiles() embed.FS {
	return embeddedLevels
}
