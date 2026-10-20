package animations

import (
	"testing"

	"gokeen/engine/mathematics"
)

func close(a, b float32) bool {
	d := a - b
	return d > -1e-4 && d < 1e-4
}

func TestSampleInterpolates(t *testing.T) {
	skel := &Skeleton{Bones: []Bone{{Name: "root", Parent: -1, Local: maths.IdentityTransform()}}}
	clip := &Clip{
		Duration: 2,
		Tracks: []BoneTrack{{
			Bone: 0,
			Position: []VectorKey{
				{Time: 0, Value: maths.V3(0, 0, 0)},
				{Time: 2, Value: maths.V3(4, 0, 0)},
			},
			Rotation: []QuatKey{
				{Time: 0, Value: maths.QIdentity()},
				{Time: 2, Value: maths.QFromAxisAngle(maths.V3(0, 1, 0), maths.Pi)},
			},
		}},
	}
	local := Sample(skel, clip, 1)
	if !close(local[0].Position.X, 2) {
		t.Fatalf("position: %+v", local[0].Position)
	}
	got := local[0].Rotation.Rotate(maths.V3(1, 0, 0))
	if !close(got.X, 0) || !close(got.Z, -1) {
		t.Fatalf("rotation: %+v", got)
	}
}

func TestSampleLoopAndClamp(t *testing.T) {
	skel := &Skeleton{Bones: []Bone{{Name: "root", Parent: -1, Local: maths.IdentityTransform()}}}
	track := []VectorKey{{Time: 0, Value: maths.V3(0, 0, 0)}, {Time: 2, Value: maths.V3(4, 0, 0)}}
	loop := &Clip{Duration: 2, Loop: true, Tracks: []BoneTrack{{Bone: 0, Position: track}}}
	if got := Sample(skel, loop, 3)[0].Position.X; !close(got, 2) {
		t.Fatalf("loop: %v", got)
	}
	clamp := &Clip{Duration: 2, Tracks: []BoneTrack{{Bone: 0, Position: track}}}
	if got := Sample(skel, clamp, 99)[0].Position.X; !close(got, 4) {
		t.Fatalf("clamp: %v", got)
	}
}

func TestBuildPoseHierarchy(t *testing.T) {
	parent := maths.IdentityTransform()
	parent.Position = maths.V3(0, 2, 0)
	child := maths.IdentityTransform()
	child.Position = maths.V3(0, 1, 0)
	skel := &Skeleton{Bones: []Bone{
		{Name: "root", Parent: -1, Local: parent, InverseBind: maths.M4Identity()},
		{Name: "arm", Parent: 0, Local: child, InverseBind: maths.M4Identity()},
	}}
	pose := BuildPose(skel, []maths.Transform{parent, child})
	got := pose.World[1].TransformPoint(maths.V3(0, 0, 0))
	if !close(got.Y, 3) {
		t.Fatalf("world child: %+v", got)
	}
}
