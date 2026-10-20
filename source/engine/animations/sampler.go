package animations

import (
	"sort"

	"gokeen/engine/mathematics"
)

// Sample evaluates a clip at the given time in seconds and returns local bone
// transforms. Missing tracks keep the skeleton bind-pose local transform.
func Sample(skeleton *Skeleton, clip *Clip, time float32) []maths.Transform {
	local := make([]maths.Transform, len(skeleton.Bones))
	for i, bone := range skeleton.Bones {
		local[i] = bone.Local
	}
	if clip == nil || len(clip.Tracks) == 0 {
		return local
	}
	if clip.Duration > 0 {
		if clip.Loop {
			time = float32(int(time*1000)%int(clip.Duration*1000)) / 1000
		} else {
			if time < 0 {
				time = 0
			}
			if time > clip.Duration {
				time = clip.Duration
			}
		}
	}
	for _, track := range clip.Tracks {
		if track.Bone < 0 || int(track.Bone) >= len(local) {
			continue
		}
		tr := local[track.Bone]
		if len(track.Position) > 0 {
			tr.Position = sampleVec3(track.Position, time)
		}
		if len(track.Scale) > 0 {
			tr.Scale = sampleVec3(track.Scale, time)
		}
		if len(track.Rotation) > 0 {
			tr.Rotation = sampleQuat(track.Rotation, time)
		}
		local[track.Bone] = tr
	}
	return local
}

func BuildPose(skeleton *Skeleton, local []maths.Transform) Pose {
	pose := NewPose(len(skeleton.Bones))
	for i, bone := range skeleton.Bones {
		tr := maths.IdentityTransform()
		if i < len(local) {
			tr = local[i]
		}
		pose.Local[i] = tr
		m := tr.Matrix()
		if bone.Parent >= 0 && int(bone.Parent) < i {
			m = pose.World[bone.Parent].Mul(m)
		}
		pose.World[i] = m
		pose.Skin[i] = m.Mul(bone.InverseBind)
	}
	return pose
}

func sampleVec3(keys []VectorKey, time float32) maths.Vec3 {
	if len(keys) == 0 {
		return maths.Vec3{}
	}
	i := sort.Search(len(keys), func(i int) bool { return keys[i].Time >= time })
	if i == 0 {
		return keys[0].Value
	}
	if i >= len(keys) {
		return keys[len(keys)-1].Value
	}
	a, b := keys[i-1], keys[i]
	span := b.Time - a.Time
	if span <= maths.Epsilon {
		return b.Value
	}
	return a.Value.Lerp(b.Value, (time-a.Time)/span)
}

func sampleQuat(keys []QuatKey, time float32) maths.Quat {
	if len(keys) == 0 {
		return maths.QIdentity()
	}
	i := sort.Search(len(keys), func(i int) bool { return keys[i].Time >= time })
	if i == 0 {
		return keys[0].Value
	}
	if i >= len(keys) {
		return keys[len(keys)-1].Value
	}
	a, b := keys[i-1], keys[i]
	span := b.Time - a.Time
	if span <= maths.Epsilon {
		return b.Value
	}
	return a.Value.Slerp(b.Value, (time-a.Time)/span)
}
