package animations

import (
	"gokeen/engine/mathematics"
)

type SkeletonID string
type ClipID string

type Bone struct {
	Name        string
	Parent      int32
	Local       maths.Transform
	InverseBind maths.Mat4
}

type Skeleton struct {
	ID    SkeletonID
	Name  string
	Bones []Bone
}

type VectorKey struct {
	Time  float32
	Value maths.Vec3
}

type QuatKey struct {
	Time  float32
	Value maths.Quat
}

type BoneTrack struct {
	Bone     int32
	Position []VectorKey
	Scale    []VectorKey
	Rotation []QuatKey
}

type Clip struct {
	ID       ClipID
	Name     string
	Duration float32
	Loop     bool
	Tracks   []BoneTrack
}

type Pose struct {
	Local []maths.Transform
	World []maths.Mat4
	Skin  []maths.Mat4
}

func NewPose(boneCount int) Pose {
	return Pose{
		Local: make([]maths.Transform, boneCount),
		World: make([]maths.Mat4, boneCount),
		Skin:  make([]maths.Mat4, boneCount),
	}
}
