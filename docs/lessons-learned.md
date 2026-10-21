# GoKeen Lessons Learned and Pitfalls

This document records hard-won implementation details, traps, and caveats from
the legacy backport and the Go/SDL2/Vulkan implementation. Read it before
changing asset conversion, skeleton handling, coordinates, or the Vulkan bridge.

---

## 1. Legacy B3D / Blitz3D

### 1.1 Endianness

- B3D is little-endian.
- 3DS is also little-endian. The original planning notes incorrectly called
  3DS big-endian; the working parser and real-file tests use little-endian.
- Do not infer endianness from the chunk-ID byte order alone. The 3DS main
  chunk starts with bytes `4D 4D`; the following length is little-endian.

### 1.2 Materials are per TRIS, not per MESH

A B3D `MESH` chunk has a master brush ID which is often `-1`. The actual
material assignment comes from each `TRIS` chunk's brush ID.

Consequence:

- Parse every `TRIS` group and resolve its brush.
- Do not assume the `MESH` brush applies to all triangles.
- Splitting static geometry into submeshes by brush is required for correct
  materials and lightmaps.

### 1.3 Lightmaps and layered materials

Legacy Level 1 uses:

- multiple UV sets,
- texture flag `0x10000` for the second UV set,
- texture blend `5` (`BLEND_MULTIPLY2`) for baked lightmaps,
- brush texture layers where the lightmap can live at a nonzero layer index.

Consequence:

- Preserve every brush layer, its texture, UV set, blend mode, and flags.
- The runtime currently supports base + lightmap; more layers are preserved but
  not rendered yet.
- Missing lightmaps such as `map2b_lm.bmp` need a deterministic fallback.

### 1.4 Quaternion convention is not Hamilton-standard

Blitz3D's `Quat` multiplication and rotation use a convention opposite to the
engine's standard Hamilton quaternions. Loading B3D quaternion values directly
produces inverted bone rotations.

Correct B3D-to-engine conversion:

1. Convert Blitz3D quaternion `(w,x,y,z)` to engine convention:
   `(w, -x, -y, -z)`.
2. Apply the engine's Z mirror (if used): `(w, -x, -y, z)` again, which becomes
   `(w, x, y, -z)` relative to the B3D source.

Symptom of getting this wrong:

- arms and legs swing sideways instead of front/back,
- poses collapse or look broken,
- head tilt is inverted.

### 1.5 Bone weights and the hidden mesh root bone

Blitz3D appends the mesh itself at index `0` of the animation bone list before
calling `createBones()`. B3D `BONE` chunks index the remaining bones starting
after that mesh root.

Consequences:

- Vertex weights may be absent for some vertices.
- Blitz3D renders unweighted vertices with **bone 0**, the mesh root.
- If unweighted vertices are left with all-zero weights in a GPU skinning
  shader, they collapse to the origin and create holes in the model.

Fix:

- Assign unweighted vertices to skeleton bone 0 with weight 1.
- Normalize the selected top four weights.
- Keep the mesh/root node as bone 0 so the root transform and attachment math
  remain valid.

### 1.6 Skeleton hierarchy and attachment points

- Include the mesh root and intermediate (non-BONE) nodes in the skeleton when
  possible. This preserves hand/head parent transforms and attachment points.
- Map B3D BONE weight indices to skeleton indices explicitly; do not assume the
  BONE order equals the complete node order.
- Use `HandR` for the gun once the skeleton conversion is correct.

### 1.7 Animation data

- Each Keen/Yorp B3D file contains its own animation sequence.
- B3D `ANIM.FPS` is often `0`; fall back to a sane rate (the importer uses 30).
- The base file's node local transforms form the bind pose.
- Clip keys override local transforms; positions, scales, and quaternions all
  need coordinate conversion.
- `KeenDie.b3d` uses a different `ROOT` + `joint1..5` rig. Mapping its nodes by
  name onto the KeenWalk skeleton produces no tracks; retarget the local joint
  deltas manually (see 1.11).
- There is no animation blending/transition controller yet.

### 1.8 Units and scales differ wildly

- B3D Keen/Yorp models are roughly 60-100 units tall.
- The static 3DS Keen model is roughly 3.8 units tall.
- Do not use one model-space scale for all sources.
- The runtime scales the skinned Keen model by about `0.036` and offsets its
  feet to the controller position.

### 1.9 Coordinate conversion and winding

- The engine uses right-handed, Y-up, `-Z` forward.
- B3D content is mirrored during import (Z is negated).
- Mirroring reverses triangle winding; static geometry must reverse its index
  order.
- Skinned geometry also requires the skeleton local transforms to be converted
  consistently. Conjugating each local transform with the mirror is the reliable
  approach.
- 3DS content is first converted with a Y/Z axis swap (Blitz3D's 3DS conversion
  matrix), then mirrored for the engine. The combined transform has positive
  determinant, so its winding handling differs from B3D.

### 1.10 KeenPogo is a static 3DS pose, not a rig

- The legacy executable loads `items\pogo.3ds` for the pogo pickup and
  `Keen3f\keenjump.b3d` for the jump; it does not load `KeenPogo.3ds`.
- `KeenPogo.3ds` is nevertheless a useful authored Keen-on-pogo pose with the
  pogo stick included.
- It has no bones. To backport it correctly, skin it to the labelled `skel_keen`:
  map materials/objects to bones (`Helmet`/`Face`/`Skin` to `Head`, `Hand` by
  side to `HandL`/`HandR`, `Watch` to `WristR`, `Shoe` by side to `FootL`/`FootR`,
  body materials to the pelvis/spine chain, pogo materials to a root-child
  `Pogo` bone).
- 3DS Keen units are about 24.5x smaller than B3D Keen units. Align the pogo mesh
  into B3D space with a uniform scale and translation before skinning.
- The pogo mesh is authored in the pogo pose, so do not play the walk/jump clips
  on it. Use an empty bind-pose clip so the feet stay planted; the hands stay on
  the handle.
- The pogo stick bottom is about 1.0 m below the feet at runtime scale. Place the
  mesh so its bounds' minimum Y is at the player position, and raise the camera
  target while deployed.

### 1.11 KeenDie is a squash shell, not a full-body rig

- `KeenDie.b3d` contains only the head/helmet/face/skin shell and shoes; it has
  no torso, arms, jeans, shirt, or watch groups.
- Its rig is `ROOT` -> `joint1` -> `joint2` -> `joint3` plus
  `joint1` -> `joint4` -> `joint5`. Vertices weighted to `joint2`/`joint4` are
  the shoes/feet; unweighted vertices bind to the mesh root, matching Blitz3D's
  `bones[0]` rule in `loader_b3d.cpp`/`surface.cpp`.
- The legacy animation only keys `joint1`, `joint2`, `joint4`, and the unweighted
  head/body does not move. Treat the mesh as a squashed death pose, not a
  full-body death animation.
- Retarget the rotation deltas rather than copying absolute local quaternions:
  `delta = srcBind^-1 * srcKey`, then `dstKey = dstBind * delta`.
- Geometry mapping used by the importer:

  | Legacy joint | Labelled bone |
  |---|---|
  | `joint1` | `PelvisC` |
  | `joint2` | `AnkleR` |
  | `joint3` | `FootR` |
  | `joint4` | `AnkleL` |
  | `joint5` | `FootL` |

- The legacy head-to-feet span is about 30% of the standing Keen span. Apply the
  Y squash to `ShoulderC`; `Head` is a child, so inverse-scale `Head` Y by
  `1/0.30` to keep the helmet/head at its original size.
- Squashing below `ShoulderC` also lifts the feet. Lower the model by
  `(1-squash) * (ShoulderC world Y - mesh min Y)` so the feet stay grounded.
- Keep the legacy `player_keen_dead` mesh/skeleton in the pack for verification;
  rendering the normal labelled Keen with the retargeted clip and squash avoids
  the missing torso.

### 1.12 Texture paths

Legacy paths include Windows drive paths, relative `../../..` paths, mixed case,
and basenames that exist in multiple folders.

- Resolve exact relative path first.
- Then resolve case-insensitive relative path.
- Then resolve by lowercase basename.
- Report ambiguous matches.
- Provide a fallback checker/average-color texture for missing files.

---

## 2. Asset Pipeline

- Legacy formats are conversion-time only. The game must never require
  `old_game` at runtime.
- Editable source assets are glTF/PNG/JSON; the runtime uses the generated
  `.gkp` pack embedded with `go:embed`.
- Generated packs are deterministic and content-hashed. Keep the writer sorted
  by asset ID.
- The source manifest does not currently include the runtime-only skinned mesh
  because skinned glTF export is not implemented. Asset checks must not assume
  every pack mesh has a glTF source.
- `make generate` runs both asset conversion and shader compilation. Without
  `glslc`, shader generation fails.
- `go generate ./assets ./shaders` depends on the legacy `old_game` directory.
  Tests that use real assets skip cleanly when it is missing.
- The `.dat` header/record layout is only partially decoded. Do not claim full
  legacy entity recovery yet; Level 1 exit and pickups are authored JSON.

---

## 3. Go + cgo + Vulkan

### 3.1 cgo pointer rules

Passing a Go struct pointer to C is only safe when the Go memory does not
contain Go pointers. Vulkan structs commonly contain nested pointers.

Symptom:

```text
panic: runtime error: argument of cgo function has Go pointer to unpinned Go pointer
```

Rule of thumb:

- Keep nested Vulkan struct setup in the C bridge (`bridge.c`).
- Pass primitives, handles, and flat Go arrays to C.
- Do not construct `Vk*CreateInfo` structs in Go when they point to Go slices or
  other Go structs.
- If C needs an array of pointers, allocate it with `C.malloc`.

### 3.2 Vulkan handles are not uniformly integer-like

- Some handles are pointer types in cgo (`VkDevice`, `VkSemaphore`, `VkFence`,
  `VkSurfaceKHR`, `VkSwapchainKHR`, etc.).
- Compare them with `nil`, not `0`.
- SDL surface creation returns an opaque pointer; keep the conversion in the C
  bridge instead of round-tripping through `uintptr`.

### 3.3 Vulkan setup details

- The swapchain needs a depth attachment and a render pass that clears depth.
- Vulkan NDC uses Y-down and depth `[0,1]`; the projection flips Y.
- Resize must destroy and recreate framebuffers, depth resources, swapchain,
  and command buffers.
- Command buffers are reset per frame; do not record one clear-only frame and
  reuse it forever.
- Use separate descriptor pools or enough capacity when adding bone palette
  descriptors; per-instance bone sets consume uniform-buffer descriptors.
- Bone palettes use a second descriptor set and a separate pipeline layout.
- The skinned vertex shader expects 64-byte vertices with JOINTS_0/WEIGHTS_0.
- sRGB swapchain output is used directly; there is no HDR/tone-mapping pass yet.

### 3.4 Shader generation

- SPIR-V is generated by `glslc` and embedded.
- Keep `shaders/*.vert`, `shaders/*.frag`, and `shaders/generated/*.spv` in sync.
- The static and skinned pipelines share the fragment shader but use different
  vertex shaders.

---

## 4. Math and Camera

- Camera yaw convention: forward = `(sin(yaw), 0, -cos(yaw))`.
- A model using `QFromAxisAngle(Y, yaw)` turns opposite to the camera. The
  character model must use `-yaw` for its body rotation.
- Mouse-right must increase camera yaw to turn right.
- Head pitch required an inverted sign relative to camera pitch because of the
  head bone's local axes.
- The skin matrix is `animatedWorld * inverseBind`; the model matrix is applied
  after skinning.
- Do not apply runtime model scale to bone-space attachment offsets twice.

---

## 5. Gameplay and Physics

- The capsule controller is a conservative depenetration implementation, not a
  full continuous collision solver.
- Current gaps: step-up, slope handling, camera spring-arm collision, precise
  ledge behavior.
- Ground detection uses a short downward ray and accepts normals with
  `Y > 0.5`.
- The fixed-step clock exists but the runtime still uses variable frame delta.
  This is a known inconsistency.
- Pogo ownership (`PogoEnabled`) and deployment (`PogoDeployed`) are separate.
  The pickup grants ownership only; Ctrl toggles deployment when owned. Space is
  the sole bounce input; pogo uses `PogoSpeed` when deployed and `JumpSpeed`
  otherwise.
- The runtime swaps to `player_keen_pogo` while deployed. The normal skinned
  Keen and gun are hidden by zero-scale transforms.
- Death keeps the normal labelled Keen mesh and plays the retargeted `die` clip.
  The clip scales `ShoulderC` Y to 0.30 and `Head` Y to 1/0.30, so the body
  squeezes but the helmet/head stays full-size. The model is lowered to keep the
  feet grounded. The pose holds for 1.2s, then `Respawn()` runs and health is
  restored. There is no game-over state when lives reach zero.
- Projectiles are separate moving entities, not hitscan tracers. They consume
  ammo, collide with geometry/enemies, and stun Yorps.
- The gun is attached to `HandR`, but the grip offset is still manually tuned.
- Exit triggers use a forgiving horizontal radius and vertical range because
  the exact legacy trigger volume is not recovered.
- Level 1 has placeholder pickups near spawn and a B3D-derived exit. Levels 2/5
  and the map still need authored entity placement.

---

## 6. SDL / Cursor Lifecycle

- Relative mouse mode hides the cursor; do not additionally call
  `SDL_ShowCursor(SDL_DISABLE)` for capture. That makes restoration harder.
- On release:
  - disable relative mode,
  - clear window/mouse grabs,
  - set the system arrow cursor,
  - show the cursor,
  - pump events.
- ESC, window close, SIGINT, and SIGTERM all need to release the mouse.
- Keep an `atexit` fallback, but guard it with an “SDL still active” flag so it
  does not call SDL after `SDL_Quit`.
- On Wayland compositors, cursor warping/visibility can still be compositor
  dependent; verify on the target desktop.

---

## 7. Verification Practices

- Run `make generate` after touching shaders, importer, or level JSON.
- Run:
  - `go test -count=1 ./...`
  - `go vet ./...`
  - `make assetcheck`
  - `make levelcheck`
- Visual checks matter for:
  - skeleton orientation and limb motion,
  - gun hand attachment,
  - projectile visibility,
  - cursor release,
  - material/lightmap appearance.
- Automated renderer screenshot testing is still pending.
- `gokeen -spawn-at-exit -frames 5` verifies the Level 1 -> Level 2 transition.
- `gokeen -autofire` is the quickest moving-projectile/gun visual check.

---

## 8. High-Risk Areas for Future Work

1. Changing quaternion conversion without re-checking walk animation and
   attachment points.
2. Introducing skinned glTF export or animation blending without preserving the
   Blitz3D bind-pose convention.
3. Changing cgo struct ownership and reintroducing nested Go pointers.
4. Moving to a fixed-step runtime without updating animation, camera, and
   projectile time handling.
5. Adding per-instance effects without expanding descriptor pools or moving to
   dynamic/storage-buffer descriptors.
6. Recovering `.dat` semantics without validating against level bounds and
   walkable surfaces.
7. Adding more Vulkan pipelines without depth, alpha, and pipeline-layout
   consistency checks.
