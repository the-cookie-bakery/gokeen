package levels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"

	"gokeen/engine/mathematics"
)

const FormatVersion = 2

type LevelID string

type Vec3 [3]float32

func (v Vec3) Vector() maths.Vec3 {
	return maths.V3(v[0], v[1], v[2])
}

func (v Vec3) IsFinite() bool {
	return v.Vector().IsFinite()
}

func (v *Vec3) UnmarshalJSON(data []byte) error {
	var raw []float32
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("expected [x, y, z]: %w", err)
	}
	if len(raw) != 3 {
		return fmt.Errorf("expected 3 components, got %d", len(raw))
	}
	*v = Vec3{raw[0], raw[1], raw[2]}
	return nil
}

func (v Vec3) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float32{v[0], v[1], v[2]})
}

// Euler stores rotation in degrees around X, Y, and Z.
type Euler [3]float32

func (e Euler) Quaternion() maths.Quat {
	return maths.QFromEulerDegrees(e[0], e[1], e[2])
}

func (e Euler) IsFinite() bool {
	return finite32(e[0]) && finite32(e[1]) && finite32(e[2])
}

func (e *Euler) UnmarshalJSON(data []byte) error {
	var raw []float32
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("expected [pitch, yaw, roll] degrees: %w", err)
	}
	if len(raw) != 3 {
		return fmt.Errorf("expected 3 components, got %d", len(raw))
	}
	*e = Euler{raw[0], raw[1], raw[2]}
	return nil
}

func (e Euler) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float32{e[0], e[1], e[2]})
}

type RGB [3]float32

func (c RGB) IsFinite() bool {
	return finite32(c[0]) && finite32(c[1]) && finite32(c[2])
}

func (c *RGB) UnmarshalJSON(data []byte) error {
	var raw []float32
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("expected [r, g, b]: %w", err)
	}
	if len(raw) != 3 {
		return fmt.Errorf("expected 3 components, got %d", len(raw))
	}
	*c = RGB{raw[0], raw[1], raw[2]}
	return nil
}

func (c RGB) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float32{c[0], c[1], c[2]})
}

type TransformSpec struct {
	Position Vec3  `json:"position"`
	Rotation Euler `json:"rotation"`
	Scale    *Vec3 `json:"scale,omitempty"`
}

func (t TransformSpec) ScaleOrOne() maths.Vec3 {
	if t.Scale == nil {
		return maths.V3(1, 1, 1)
	}
	return t.Scale.Vector()
}

func (t TransformSpec) Transform() maths.Transform {
	return maths.Transform{
		Position: t.Position.Vector(),
		Rotation: t.Rotation.Quaternion(),
		Scale:    t.ScaleOrOne(),
	}
}

func (t TransformSpec) IsFinite() bool {
	if !t.Position.IsFinite() || !t.Rotation.IsFinite() {
		return false
	}
	if t.Scale != nil && !t.Scale.IsFinite() {
		return false
	}
	return true
}

type Metadata struct {
	Author string            `json:"author,omitempty"`
	Notes  string            `json:"notes,omitempty"`
	Tags   []string          `json:"tags,omitempty"`
	Extras map[string]string `json:"extras,omitempty"`
}

type Sun struct {
	Direction Vec3    `json:"direction"`
	Color     RGB     `json:"color"`
	Intensity float32 `json:"intensity"`
}

type Fog struct {
	Color RGB     `json:"color"`
	Start float32 `json:"start"`
	End   float32 `json:"end"`
}

type Environment struct {
	Ambient RGB    `json:"ambient"`
	Sun     *Sun   `json:"sun,omitempty"`
	Fog     *Fog   `json:"fog,omitempty"`
	Skybox  string `json:"skybox,omitempty"`
}

type GeometryInstance struct {
	ID        string        `json:"id"`
	Mesh      string        `json:"mesh"`
	Collision string        `json:"collision,omitempty"`
	Transform TransformSpec `json:"transform"`
	Visible   *bool         `json:"visible,omitempty"`
}

type Spawn struct {
	Position Vec3  `json:"position"`
	Rotation Euler `json:"rotation"`
}

type EntitySpec struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Transform  TransformSpec   `json:"transform"`
	Properties json.RawMessage `json:"properties,omitempty"`
	Editor     *EditorMeta     `json:"editor,omitempty"`
}

type EditorMeta struct {
	Layer  string            `json:"layer,omitempty"`
	Group  string            `json:"group,omitempty"`
	Locked bool              `json:"locked,omitempty"`
	Hidden bool              `json:"hidden,omitempty"`
	Extras map[string]string `json:"extras,omitempty"`
}

type LightType string

const (
	LightDirectional LightType = "directional"
	LightPoint       LightType = "point"
	LightSpot        LightType = "spot"
)

type LightSpec struct {
	ID        string        `json:"id"`
	Type      LightType     `json:"type"`
	Transform TransformSpec `json:"transform"`
	Color     RGB           `json:"color"`
	Intensity float32       `json:"intensity"`
	Range     float32       `json:"range,omitempty"`
	InnerCone float32       `json:"inner_cone_degrees,omitempty"`
	OuterCone float32       `json:"outer_cone_degrees,omitempty"`
	Editor    *EditorMeta   `json:"editor,omitempty"`
}

type TriggerSpec struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Transform  TransformSpec   `json:"transform"`
	Size       *Vec3           `json:"size,omitempty"`
	Radius     float32         `json:"radius,omitempty"`
	Properties json.RawMessage `json:"properties,omitempty"`
	Editor     *EditorMeta     `json:"editor,omitempty"`
}

type Level struct {
	FormatVersion int                `json:"format_version"`
	ID            LevelID            `json:"id"`
	Name          string             `json:"name"`
	Metadata      Metadata           `json:"metadata,omitempty"`
	Environment   Environment        `json:"environment"`
	Geometry      []GeometryInstance `json:"geometry"`
	Spawn         Spawn              `json:"spawn"`
	Entities      []EntitySpec       `json:"entities"`
	Lights        []LightSpec        `json:"lights"`
	Triggers      []TriggerSpec      `json:"triggers"`
}

func (l *Level) UnmarshalJSON(data []byte) error {
	type alias Level
	var a alias
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return err
	}
	*l = Level(a)
	return nil
}

func finite32(v float32) bool {
	f := float64(v)
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
