# GoKeen Architecture

## Overview

GoKeen is split into a reusable engine, a typed gameplay/entity layer, editable
content data, and offline conversion tools. The current executable uses the SDL2
and Vulkan packages directly; extracting backend-neutral interfaces is planned.

```text
cmd/gokeen
    |
    +-- engine/platforms/sdl2       window, events, surface
    +-- engine/renderers/vulkan     rendering backend
    +-- gameplay                    player/enemy/pickup/trigger rules
    +-- entities                    typed ECS (scaffolding, not runtime yet)
    +-- levels                      JSON levels
    +-- assets                      embedded runtime pack
```

## Implementation Status

The architecture below describes both the current implementation and intended
structure. This table is the quick reference for what actually exists.

| Subsystem | Current state |
|---|---|
| Mathematics | Implemented |
| Fixed-step clock | Implemented, not yet driving the runtime loop |
| Typed ECS | Implemented and tested; not yet used by the main gameplay loop |
| Asset pack/store/embed | Implemented |
| B3D/3DS/`.dat` parsing | Implemented; `.dat` semantics partial |
| Static glTF export | Implemented |
| Skinned glTF export | Not implemented |
| Level JSON/validation | Implemented |
| Entity registry | Implemented; runtime spawning is still an ad-hoc switch |
| SDL2 window/events | Implemented |
| SDL2 gamepad/audio | Not implemented |
| Vulkan static mesh rendering | Implemented |
| Vulkan skinned rendering | Implemented |
| Vulkan sprite/UI rendering | Not implemented |
| Blinn-Phong + lightmap | Implemented |
| HDR/post/shadows | Not implemented |
| BVH/raycast/capsule | Implemented; movement tuning partial |
| Player controller | Implemented |
| Pogo | Implemented |
| Laser + ammo | Implemented (moving projectiles) |
| Yorp actor/stun | Implemented, static model only |
| Keen skinned animation | Implemented: walk/jump/shoot/pogo/die clips |
| Cursor lifecycle | Implemented: release on exit/signals |
| Pickups/score/key/health | Implemented for a subset of items |
| Exit/level transition | Implemented for Level 1 -> Level 2 |
| HUD | Window-title debug HUD only |
| Audio | Not implemented |
| Asset viewer/editor | Level fly viewer only; no dedicated editor |

## Directory Layout

```text
source/
  cmd/
    gokeen/                  runnable game
    gokeen-assetconv/        legacy -> glTF/JSON/pack
    gokeen-assetcheck/       pack/glTF/level verification
    gokeen-levelcheck/       level JSON verification
  engine/
    animations/              skeleton/clip sampling
    core/                    wall clock + fixed-step clock
    mathematics/             vectors, matrices, quaternions, rays, AABB
    physics/                 BVH, raycasts, capsule contact/movement
    platforms/sdl2/          cgo window/event/surface/title bridge
    renderers/vulkan/        cgo Vulkan renderer
  entities/
    components/              typed component structs
    ids.go, store.go, world.go
  gameplay/
    cameras/                 third-person camera
    enemies/                 Yorp actor and stun state
    pickups/                 item effects and player inventory updates
    players/                 player controller and state
    triggers/                trigger volume helper
  levels/
    data/                    embedded level JSON
    format/loader/registry/validation/backport manifest
  assets/
    pack/                    generated embedded `.gkp`
    sources/                 editable glTF, PNG, source manifest
  shaders/                   GLSL + embedded SPIR-V
  tools/legacy/
    b3d/ threeds/ leveldat/ textures/ gltf/ verify/ importer/
```

Planned packages that do not exist yet are intentionally omitted here.

## Runtime Types

### Mathematics

All engine code uses a single coordinate convention:

- Right-handed.
- Y up.
- `-Z` forward.
- Meters as world units.
- Matrices are column-major for Vulkan.

The legacy importer converts Blitz3D/Direct3D-style coordinates and triangle
winding before writing glTF. The conversion is centralised in
`tools/legacy` and covered by bounds/winding tests.

Core types:

- `Vec2`, `Vec3`, `Vec4`.
- `Quat`.
- `Mat4`.
- `AABB`.
- `Ray`.
- `Transform`.

### Entity Component System

`source/entities` is a small generational ECS:

- `EntityID` is a generational handle, not a raw index.
- Components are plain Go structs.
- Each component type has a typed store.
- Queries iterate the smallest alive store and check the others.
- Entity destruction removes components without reusing stale IDs.

The ECS is implemented and tested. It is not yet the runtime gameplay store;
current systems use plain controller structs in `gameplay/`. Migrating the
runtime to ECS-backed entities is planned.

Gameplay-specific component definitions live under `source/entities/components`.

### Assets

Asset IDs are typed strings:

```go
type MeshID string
type TextureID string
type MaterialID string
type SkeletonID string
type ClipID string
type AudioClipID string
```

The runtime asset store exposes typed lookups. Missing IDs are errors, not
zero-value fallbacks.

The embedded pack is generated offline. Its header contains:

- Magic.
- Format version.
- Content hash.
- Section table.

Sections contain meshes, materials, textures, skeletons, clips, and audio clips.
Strings are stored inline in their sections. The reader validates offsets,
limits, hashes, and cross-references before exposing typed lookups.

### Levels

A level is a typed JSON document:

- Environment and lighting.
- Geometry instances referencing mesh assets.
- Spawn point.
- Entities with typed properties.
- Triggers.
- Lights.
- Optional editor metadata.

The loader:

1. Strictly decodes known fields.
2. Rejects unknown runtime fields.
3. Resolves asset references.
4. Validates IDs, transforms, bounds, and entity properties.
5. Produces a runtime `Level` value.

A typed entity registry exists with definitions for type name, display metadata,
property decoding, and spawn metadata. The current runtime still spawns
`yorp`, `pickup`, and `exit` through a switch in `cmd/gokeen`, so registry-driven
spawning and full property validation remain planned.

## Platform Layer

`engine/platforms/sdl2` currently contains a minimal cgo bridge:

- `SDL_Init` / `SDL_Quit`.
- Window creation with `SDL_WINDOW_VULKAN`.
- Vulkan instance extension enumeration.
- Vulkan surface creation.
- Event polling (quit, key, mouse move/button/wheel, resize).
- Relative mouse mode.
- Window title updates.

Planned but not implemented:

- Game controller discovery/events.
- Audio device creation and queued PCM output.

No Vulkan rendering is performed by SDL. SDL only supplies the platform surface.

The platform package exposes engine-owned structs and enums. SDL handles do not
leak into the rest of the engine.

## Renderer

There is currently no backend-neutral renderer interface package. `cmd/gokeen`
uses `engine/renderers/vulkan` directly. Introducing a renderer interface is
planned before adding another backend.

`engine/renderers/vulkan` currently implements:

- Instance and physical/logical device selection.
- Swapchain and depth target.
- Frame synchronization (one frame in flight).
- Buffers and textures (host-visible uploads).
- Per-material descriptor sets and per-instance bone buffers.
- Static and skinned mesh pipelines.
- Command buffers and resize/recreation.
- Basic resource destruction.

Planned but not implemented:

- Validation layers/debug messenger.
- Staging-buffer uploads and device-local vertex/index buffers.
- Alpha, sprite, skybox, and UI pipelines.
- Multi-frame-in-flight synchronization.
- HDR target, tone mapping, FXAA, bloom, SSAO.
- Shadow maps and frustum culling.

### Pipelines

Current:

- Opaque static mesh with depth testing.
- Skinned opaque mesh with a 64-bone palette.

Planned:

- Alpha-test mesh.
- Alpha-blend mesh.
- Additive sprite/effect.
- Skybox.
- UI.

### Shading

The current shader is per-pixel Blinn-Phong:

```text
ambient * lightmap
+ diffuse(N,L) * lightColor * lightmap
+ BlinnPhong(N,H,shininess) * lightColor * lightmap
```

The renderer draws directly into the sRGB swapchain. There is no HDR target,
tone mapping, or gamma composite pass yet. Textures are sampled as sRGB.

### Material Layers

The importer converts legacy brush layers into a runtime material:

- Base colour and alpha.
- Layer 0..N textures.
- Per-layer UV set.
- Per-layer blend mode.
- Alpha mode.
- Double-sided flag.
- Emissive.
- Lightmap reference.

The runtime currently supports a base texture plus a lightmap/multiply layer.
Additional layers are preserved in the asset data but are not yet rendered.
Missing textures use a generated checker fallback.

## Animation

The animation package provides:

- `Skeleton`: ordered bones with parent indices and inverse bind matrices.
- `Clip`: per-bone position, scale, and rotation tracks.
- `Sampler`: linear position/scale and slerp rotation.
- `Pose`: current local and world bone transforms.

Sampling is implemented and tested. The importer emits the Keen skeleton and
walk/jump/shoot clips into the pack, and the Vulkan renderer has a skinned
pipeline with a per-instance bone palette. Clip switching in the runtime is a
small state selection with no blending or cross-fade yet.

`KeenPogo.3ds` is imported as a second skinned mesh on the same labelled Keen
skeleton. Its vertices are weighted rigidly by material: helmet/face/skin to
`Head`, hands to `HandL`/`HandR`, the watch to `WristR`, shoes to the foot
bones, jeans/shirt/white to the body chain, and the pogo stick itself to a
root-child `Pogo` bone. The pogo mesh keeps the authored pogo pose at bind; an
empty bind-pose clip keeps the feet planted while the hands remain on the
handle.
`KeenDie.b3d` is a squashed head+feet death shell with a `ROOT`+`joint1..5`
rig. Its joint rotation deltas are retargeted onto the labelled skeleton:
`joint1` to `PelvisC`, `joint2` to `AnkleR`, `joint3` to `FootR`, `joint4` to
`AnkleL`, and `joint5` to `FootL`. The death clip adds a Y scale of about 0.30
on `ShoulderC` and an inverse Y scale on `Head`, so the body squeezes together
while the helmet/head keeps its original size. The runtime lowers the model by
the resulting foot lift to keep the feet grounded, plays the retargeted `die`
clip once, and holds the final pose for the death timer. The legacy
`player_keen_dead` mesh and skeleton remain in the pack for reference/verification.

## Physics

Collision geometry is converted to a triangle soup and a BVH at runtime from
the imported collision mesh. Physics currently provides:

- `Raycast`.
- `OverlapSphere`.
- Capsule contact generation.
- Conservative substepped capsule movement with depenetration.
- Downward ground query in the player/enemy controllers.

Current player controller:

- Gravity.
- Ground detection.
- Air control.
- Jump and pogo bounce.
- Fall respawn.

Not yet tuned/implemented:

- Step-up.
- Slope limit.
- Ground snap/interpolation.
- Moving platforms.
- Camera spring-arm collision.

The fixed-step clock exists and is tested, but the main loop currently uses a
variable per-frame delta. Migrating gameplay to the fixed tick is pending.

## Gameplay

The current runtime lives in `cmd/gokeen`. Each level is loaded and played in a
session loop:

1. Load embedded level JSON and validate assets.
2. Upload static geometry and materials to Vulkan.
3. Build a runtime collision BVH from the collision mesh.
4. Spawn the player, Yorps, pickups, exits, and laser effect instance.
5. Poll input and update the player/enemies/pickups.
6. Resolve exit triggers and transition to the next level.

The typed ECS and fixed-step clock exist as tested scaffolding but are not yet
wired into this loop. There is no scene manager yet. The current gameplay uses
plain controller structs in `gameplay/players`, `gameplay/enemies`, and
`gameplay/pickups`.

Implemented game flow:

- Level 1 spawn, pogo/gun pickups, hand-held gun, moving projectile laser,
  Yorp stun, exit trigger.
- Skinned Keen walk/jump/shoot animation.
- Level 1 -> Level 2 transition.
- Persistent health, ammo, score, lives, key, and pogo state across levels.
- Debug window-title HUD.

Pending game flow:

- Title/options/controls/pause/game-over states.
- HUD/stats art.
- Yorp animation, death, and impact effects.
- Animation blending and death clip wiring.
- Level 2/5/map entity placement and exits.
- Garg/PatPat.
- Audio events.
## Audio

Not implemented yet. SDL audio is initialised, but no audio device, mixer, or
PCM queue exists. The intended design remains: Go-generated procedural SFX and
pack-loaded WAV clips mixed by Go and queued through SDL.

## Offline Tools

### gokeen-assetconv

Converts legacy content into editable glTF/PNG/JSON and then packs runtime data.
It never edits existing level JSON without `--force`.

### gokeen-assetcheck

Verifies structural integrity:

- glTF index bounds.
- Vertex/triangle counts.
- Bounds.
- UV sets.
- Material and texture references.
- Animation tracks.
- Pack round-trip.

### gokeen-levelcheck

Validates level JSON against the asset manifest. Entity-type validation is
currently limited because the runtime does not yet use the registry for all
spawns.

### gokeen-assetview

Not implemented as a separate tool. `gokeen -level <id> -fly` currently serves as
a basic fly-camera level viewer. A dedicated model/animation viewer and future
editor viewport remain planned.

## Testing

- Unit tests live next to packages.
- Testdata is small and checked in.
- Legacy parser tests use both synthetic chunks and selected real assets.
- Import verification uses structural invariants rather than pixel-perfect
  comparisons.
- Renderer verification is currently manual (SDL window + screenshot); there
  is no automated Vulkan pixel/smoke test yet.
- Every implementation pass ends with `gofmt`, `go vet`, and `go test ./...`.

## Error Handling

- Parsers validate chunk lengths and offsets before reading.
- Asset loaders return contextual errors with the asset ID and section.
- Level validation reports all errors, not just the first one.
- Runtime asset lookup failures are fatal during level load.
- Unsupported legacy features are reported during conversion rather than ignored.
