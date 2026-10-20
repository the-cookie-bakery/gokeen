# GoKeen Task Tracker

This file is the working backlog and verification record. A task is only marked
verified when its tests pass and the result has been checked with the tools in
the `Tests` column.

Status values in the `Verified` column:

- `no` - not implemented, not tested, or tests failing.
- `partial` - implemented and some tests pass, but acceptance criteria are incomplete.
- `yes` - implemented, tests pass, and the milestone acceptance check succeeds.
- `n/a` - not applicable (documentation or investigation task).

## M0 - Legacy Archaeology and Conversion Specification

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M0-01 | Inventory all legacy assets and referenced paths | `go test ./tools/legacy/textures` + conversion inventory report | partial | Missing references must be reported. |
| M0-02 | Decode `.dat` record layout, axis order, and entity semantics | Unit tests against real `level1.dat`/`level2.dat` plus bounds validation | partial | Header appears to be a count/index; axis mapping must be proven. |
| M0-03 | Recover item, exit, and map-entrance placement data | Level JSON validation + manual asset viewer check | no | `.dat` appears to contain enemy spawns only. |
| M0-04 | Define coordinate conversion and triangle winding rules | Math/import bounds and winding tests | partial | Convert Blitz3D/Direct3D style to right-handed Y-up. |
| M0-05 | Define B3D -> glTF material/animation mapping | Asset conversion tests and glTF verification report | partial | Layers, lightmaps, skins, clips. |
| M0-06 | Seed `source/levels/backport.json` | `go test ./levels` | yes | Include missing Levels 3, 4, and 16. |
| M0-07 | Define import verification criteria | `gokeen-assetcheck` report | yes | Counts, bounds, UVs, tracks, textures. |

## M1 - Platform and Renderer Bring-up

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M1-01 | Go module and package scaffold | `go test ./...` | yes | Module is `gokeen`. |
| M1-02 | Core math types and coordinate convention | `go test ./engine/mathematics` | yes | Vec/Mat/Quat/AABB/Ray/Transform. |
| M1-03 | Core app lifecycle and clock | Unit tests for fixed-step accumulator | yes | 120 Hz simulation. |
| M1-04 | SDL2 cgo bridge: init, window, events | Manual window smoke test + build | yes | Linux first. |
| M1-05 | SDL2 input bridge | Build + event smoke test | partial | Keyboard, mouse, gamepad. |
| M1-06 | SDL2 audio device and queued PCM | Build + audible smoke test | no | Mixer later. |
| M1-07 | Vulkan instance/device/swapchain | Build + runtime smoke test with validation layers | yes | Hand-written cgo. |
| M1-08 | Vulkan clear screen and resize | Runtime visual check | partial | First rendered frame. |
| M1-09 | Game executable with platform loop | `go build ./cmd/gokeen` | yes | Window opens and closes cleanly. |

## M2 - Asset Pipeline and Static Level Viewer

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M2-01 | Typed asset IDs and runtime types | `go test ./assets` | yes | No untyped handles in gameplay. |
| M2-02 | Deterministic `.gkp` pack writer/reader | Round-trip and malformed-input tests | yes | Section offsets and hash validation. |
| M2-03 | Typed asset store and manifest | `go test ./assets` | yes | Source manifest + typed store. |
| M2-04 | B3D parser | Synthetic + real Level1/model tests | yes | Chunks, nodes, meshes, materials, bones, keys. |
| M2-05 | 3DS parser | Synthetic + real item tests | yes | Big-endian chunks, materials, UVs, transforms. |
| M2-06 | Texture resolver and BMP/PNG conversion | Unit tests for Windows paths, case, missing files | yes | Report ambiguities and fallbacks. |
| M2-07 | B3D -> glTF writer | Verification report | partial | Static and skinned meshes, clips, materials. |
| M2-08 | 3DS -> glTF writer | Verification report | yes | Static 3DS models converted to glTF/pack; animation support still pending. |
| M2-09 | `.dat` importer | Golden tests against known records | partial | Depends on M0-02. |
| M2-10 | Level JSON schema and strict loader | `go test ./levels` | yes | Unknown fields rejected. |
| M2-11 | Entity registry and property decoding | `go test ./entities/... ./levels` | partial | Shared with editor schema. |
| M2-12 | Level validation | `go test ./levels` | yes | IDs, assets, transforms, bounds. |
| M2-13 | `gokeen-assetconv` CLI | `go run ./cmd/gokeen-assetconv --help` + conversion test | yes | Inventory/convert/verify modes. |
| M2-14 | `gokeen-assetcheck` CLI | `go run ./cmd/gokeen-assetcheck` | yes | Verifies imports and pack. |
| M2-15 | Asset viewer loads a level/model | Manual visual verification | partial | `gokeen -level <id>` fly viewer works; dedicated model/animation viewer pending. |
| M2-16 | Static Level 1 renders with textures and lightmaps | Visual check + renderer smoke test | yes | Vulkan mesh pipeline, depth buffer, layered material descriptors. |

## M3 - Modern Lighting and Materials

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M3-01 | Blinn-Phong material shader | Shader compile + visual test | yes | Ambient/diffuse/specular implemented; emissive pending. |
| M3-02 | Lightmap multiply layer | Visual test on Level 1 | partial | UV1 lightmap path implemented; broader visual validation pending. |
| M3-03 | Dynamic directional/point/spot lights | Unit test light parsing + visual test | partial | Directional sun from level JSON is active; point/spot lights pending. |
| M3-04 | HDR offscreen target | Renderer smoke test | no | sRGB/linear correctness. |
| M3-05 | Tone mapping and gamma composite | Shader compile + visual test | no | ACES-style curve. |
| M3-06 | Alpha-test, alpha-blend, additive pipelines | Visual test | no | Signs, glass, lasers. |
| M3-07 | FXAA | Visual test | no | MSAA optional. |
| M3-08 | Skybox and fog | Visual test | no | Map and level backgrounds. |
| M3-09 | Blob shadows | Visual test | no | Character grounding. |
| M3-10 | Directional shadow maps | Visual test + performance check | no | Polish stage. |
| M3-11 | Optional SSAO/bloom | Visual test | no | Post vertical slice. |
| M3-12 | Normal mapping support | Visual test | no | Where material metadata exists. |

## M4 - Player, Physics, Camera

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M4-01 | Triangle BVH construction and query | `go test ./engine/physics` | yes | Raycast, sphere cast, overlap. |
| M4-02 | Capsule sweep and slide | `go test ./engine/physics` | partial | Swept capsule movement and landing work; full step-up/slope tuning pending. |
| M4-03 | Ground detection and snapping | `go test ./engine/physics` | yes | Downward ground query implemented and tested. |
| M4-04 | Fixed timestep simulation | `go test ./engine/core` | yes | Deterministic accumulator. |
| M4-05 | Player movement and jump | Gameplay simulation tests | yes | Camera-relative movement and jump covered by tests. |
| M4-06 | Third-person spring-arm camera | Camera collision tests + visual | partial | Third-person camera works; spring-arm collision pending. |
| M4-07 | Skeleton/clip animation sampler | `go test ./engine/animations` | yes | Loop, one-shot, transition. |
| M4-08 | Skinned mesh rendering | Visual test | no | Bone palette. |
| M4-09 | Keen model and walk/jump animations | Asset viewer + gameplay test | partial | Static Keen model and player draw work; skeletal clips pending. |
| M4-10 | Respawn and death animation | Gameplay test | no | Lives/health integration. |

## M5 - Level 1 Vertical Slice

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M5-01 | Pogo pickup and equipping | Gameplay test | no | Pickup -> HUD state. |
| M5-02 | Pogo bounce and chained bounce | Trajectory tests | yes | Pogo bounce is implemented and tested; pickup integration pending. |
| M5-03 | Laser firing and ammo | Gameplay test | partial | Hitscan laser and visible beam work; ammo/cooldowns are debug-only. |
| M5-04 | Static impact effects | Visual test | no | Zap/Zot sprites. |
| M5-05 | Yorp AI patrol/chase | AI simulation tests | partial | Yorp actor movement/landing and hit detection exist; richer AI pending. |
| M5-06 | Yorp stun with `YorpConk` | Gameplay test + animation check | partial | Laser stun state and enemy stop behavior work; `YorpConk` animation pending. |
| M5-07 | Yorp death with `YorpDie` | Gameplay test | no | Repeated hits or stomp rule. |
| M5-08 | Pickup items and score | Gameplay test | no | Items from legacy assets. |
| M5-09 | Health, ammo, lives, and blue key | Gameplay test | no | HUD state. |
| M5-10 | Exit trigger and level transition | Gameplay test | no | Target level from JSON. |
| M5-11 | HUD and stats art | Visual test | no | Use old GUI textures. |
| M5-12 | Procedural sound effects and mixer | Audible smoke test | no | Laser, pogo, pickup, hurt. |
| M5-13 | Level 1 completable | Replay test + manual playthrough | no | Milestone acceptance. |

## M6 - Full Backport Pass

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M6-01 | Level 2 import and placement | Level validation + playthrough | no | `Level2ba.b3d`, `level2.dat`. |
| M6-02 | Level 5 import and manual pogo placement | Level validation + playthrough | no | No `.dat`. |
| M6-03 | Map hub import and level entrances | Level validation + playthrough | no | `map2b.b3d`. |
| M6-04 | Garg AI and animations | Gameplay test | no | Walk/think/die/dead. |
| M6-05 | PatPat support | Gameplay test | no | Optional enemy. |
| M6-06 | Title and controls screens | Visual test | no | Old GUI art. |
| M6-07 | Options: fullscreen/windowed, resolution, audio | Settings tests | no | Persisted config. |
| M6-08 | Pause and game over | State tests | no | Scene transitions. |
| M6-09 | Update backport manifest statuses | `gokeen-levelcheck` | no | validated/playable. |
| M6-10 | Verify all assets embedded and runtime self-contained | Build test without `old_game` | no | Single binary. |

## M7 - Polish and Editor Readiness

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M7-01 | Performance profiling and culling | Frame timing report | no | Target 60 FPS at 1080p. |
| M7-02 | Spatial chunking and batching | Renderer stats test | no | Large map. |
| M7-03 | Save/settings persistence | Unit tests | no | XDG config path. |
| M7-04 | Level JSON round-trip preservation | `go test ./levels` | no | Editor metadata and unknown fields. |
| M7-05 | Entity editor schema exposure | `go test ./entities/...` | no | Drives future editor UI. |
| M7-06 | Asset viewer evolves into editor viewport | Manual check | no | Selection, transform gizmos later. |
| M7-07 | Reproducible conversion/report output | Hash comparison test | no | No timestamps in generated output. |
| M7-08 | Cross-platform build tags and docs | Linux build + cross-compile check | no | Windows/macOS later. |
| M7-09 | Final `gofmt`, `go vet`, `go test ./...` | Commands pass | yes | Release gate. |
