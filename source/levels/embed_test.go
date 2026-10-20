package levels

import (
	"testing"

	"gokeen/assets"
)

func TestEmbeddedLevelsLoad(t *testing.T) {
	pack, err := assets.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	store, err := assets.NewStore(pack)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []LevelID{"map", "level1", "level2", "level5"} {
		lvl, err := LoadEmbedded(id, StoreResolver{Store: store}, nil)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if lvl.ID != id || len(lvl.Geometry) != 1 {
			t.Fatalf("%s: %+v", id, lvl)
		}
	}
}
