package levels

import (
	"os"
	"testing"
)

func TestBackportManifest(t *testing.T) {
	data, err := os.ReadFile("backport.json")
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadBackport(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Levels) != 7 {
		t.Fatalf("levels: %d", len(m.Levels))
	}
	if m.Levels[0].ID != "map" || m.Levels[0].Status != BackportImported {
		t.Fatalf("first entry: %+v", m.Levels[0])
	}
}

func TestBackportRejectsInvalidStatus(t *testing.T) {
	data := []byte(`{"format_version":1,"levels":[{"id":"x","json_path":"data/x.json","status":"finished"}]}`)
	if _, err := LoadBackport(data); err == nil {
		t.Fatal("expected invalid status error")
	}
}
