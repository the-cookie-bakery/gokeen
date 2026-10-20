package levels

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

type BackportStatus string

const (
	BackportPending       BackportStatus = "pending"
	BackportImported      BackportStatus = "imported"
	BackportValidated     BackportStatus = "validated"
	BackportPlayable      BackportStatus = "playable"
	BackportPolished      BackportStatus = "polished"
	BackportMissingSource BackportStatus = "missing-source"
)

func (s BackportStatus) Valid() bool {
	switch s {
	case BackportPending, BackportImported, BackportValidated, BackportPlayable, BackportPolished, BackportMissingSource:
		return true
	default:
		return false
	}
}

type BackportEntry struct {
	ID         LevelID        `json:"id"`
	JSONPath   string         `json:"json_path,omitempty"`
	LegacyMesh string         `json:"legacy_mesh,omitempty"`
	EntityData string         `json:"entity_data,omitempty"`
	Status     BackportStatus `json:"status"`
	Notes      string         `json:"notes,omitempty"`
}

type BackportManifest struct {
	FormatVersion int             `json:"format_version"`
	Levels        []BackportEntry `json:"levels"`
}

func LoadBackport(data []byte) (*BackportManifest, error) {
	var m BackportManifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("decode backport manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *BackportManifest) Validate() error {
	if m == nil {
		return errors.New("nil backport manifest")
	}
	if m.FormatVersion != 1 {
		return fmt.Errorf("unsupported backport manifest version %d", m.FormatVersion)
	}
	seen := make(map[LevelID]bool)
	for i, e := range m.Levels {
		if e.ID == "" {
			return fmt.Errorf("backport entry %d has empty id", i)
		}
		if seen[e.ID] {
			return fmt.Errorf("duplicate backport level id %q", e.ID)
		}
		seen[e.ID] = true
		if !e.Status.Valid() {
			return fmt.Errorf("backport level %q has invalid status %q", e.ID, e.Status)
		}
		if e.Status != BackportMissingSource && e.JSONPath == "" {
			return fmt.Errorf("backport level %q has no json_path", e.ID)
		}
	}
	return nil
}
