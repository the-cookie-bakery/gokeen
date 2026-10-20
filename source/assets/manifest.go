package assets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const SourceManifestVersion = 1

type SourceManifest struct {
	FormatVersion int           `json:"format_version"`
	Assets        []SourceAsset `json:"assets"`
}

type SourceAsset struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Path         string `json:"path"`
	LegacySource string `json:"legacy_source,omitempty"`
	MeshID       string `json:"mesh_id,omitempty"`
}

func LoadSourceManifest(data []byte) (*SourceManifest, error) {
	var m SourceManifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("decode source manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *SourceManifest) Validate() error {
	if m == nil {
		return errors.New("nil source manifest")
	}
	if m.FormatVersion != SourceManifestVersion {
		return fmt.Errorf("unsupported source manifest version %d", m.FormatVersion)
	}
	seen := make(map[string]bool)
	for i, a := range m.Assets {
		if a.ID == "" {
			return fmt.Errorf("source asset %d has empty id", i)
		}
		if seen[a.ID] {
			return fmt.Errorf("duplicate source asset id %q", a.ID)
		}
		seen[a.ID] = true
		if a.Kind == "" || a.Path == "" {
			return fmt.Errorf("source asset %q is missing kind or path", a.ID)
		}
	}
	return nil
}

func (m *SourceManifest) Marshal() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
