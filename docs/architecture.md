# GoKeen Architecture

## Overview

GoKeen is split into a reusable engine, a typed gameplay/entity layer, editable
content data, and offline conversion tools. Runtime rendering and platform code
are isolated so gameplay does not depend on SDL or Vulkan types.

```text
cmd/gokeen
    |
    +-- engine/platforms/sdl2       window, input, audio
    +-- engine/renderers/vulkan     rendering backend
    +-- engine/scenes               state machine
    +-- gameplay                    game rules
    +-- entities                    typed ECS
    +-- levels                      JSON levels
    +-- assets                      embedded runtime pack
```

## Directory Layout

```text
source/
  cmd/
    gokeen/
    gokeen-assetconv/
    gokeen-assetcheck/
    gokeen-assetview/
    gokeen-levelcheck/
  engine/
    animations/
    audio/
    core/
    inputs/
    mathematics/
    physics/
    platforms/sdl2/
    renderers/
      vulkan/
    scenes/
    uis/
  entities/
    components/
    prefabs/
    queries/
    stores/
  gameplay/
    cameras/
    enemies/
    pickups/
    players/
    systems/
    triggers/
    weapons/
  levels/
    data/
  assets/
    pack/
    sources/
  shaders/
  tools/
    legacy/
      b3d/
      threeds/
      leveldat/
      textures/
      gltf/
      verify/
  configs/
```

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
- Systems run in a defined order from `gameplay/systems`.

Gameplay-specific components live under `source/entities/components`, while
systems that interpret them live under `source/gameplay`.

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
- String table.

Sections contain meshes, materials, textures, skeletons, clips, collision data,
and audio clips. The reader validates all offsets and limits before allocating.

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

Entity types are registered in a registry. Each definition provides:

- Type name.
- Display metadata for tools/editor.
- Property decoder.
- Spawn function.
- Default property value.

The registry is the single source of truth for JSON validation, runtime spawning,
and future editor property forms.

## Platform Layer

`engine/platforms/sdl2` contains a minimal cgo bridge:

- `SDL_Init` / `SDL_Quit`.
- Window creation with `SDL_WINDOW_VULKAN`.
- Vulkan instance extension enumeration.
- Vulkan surface creation.
- Event polling.
- Relative mouse mode.
- Game controller discovery.
- Audio device creation and queued PCM output.

No Vulkan rendering is performed by SDL. SDL only supplies the platform surface.

The platform package exposes engine-owned structs and enums. SDL handles do not
leak into the rest of the engine.

## Renderer

`engine/renderers` defines backend-neutral interfaces:

- Renderer lifetime.
- Frame begin/end.
- Resize.
- Camera.
- Draw submission.
- Mesh/material/texture handles.

`engine/renderers/vulkan` implements the backend:

- Instance and validation layers.
- Physical/logical device selection.
- Swapchain and depth target.
- Frame synchronization.
- Staging uploads.
- Buffers, images, samplers.
- Descriptor sets.
- Pipelines.
- Command buffers.
- Resize/recreation.
- Deferred destruction.

### Pipelines

- Opaque static mesh.
- Skinned opaque mesh.
- Alpha-test mesh.
- Alpha-blend mesh.
- Additive sprite/effect.
- Skybox.
- UI.

### Shading

The baseline is per-pixel Blinn-Phong:

```text
ambient + lightmap
+ sum(diffuse(N,L) * lightColor)
+ sum(BlinnPhong(N,H,shininess) * specular)
+ emissive
```

Textures are sampled in sRGB and converted to linear space. The scene renders to
an HDR target. A composite pass applies exposure, tone mapping, gamma, and FXAA.

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

The renderer initially supports base + lightmap and falls back to a documented
checker material for unsupported layer combinations.

## Animation

The animation package provides:

- `Skeleton`: ordered bones with parent indices and inverse bind matrices.
- `Clip`: per-bone position, scale, and rotation tracks.
- `Sampler`: linear position/scale and slerp rotation.
- `Controller`: loop, one-shot, transition, and speed control.
- `Pose`: current local and world bone transforms.

The Vulkan skinned pipeline receives a bone matrix palette. Models without skin
data use the static pipeline.

## Physics

Static collision geometry is converted to a triangle soup and a BVH offline.
Runtime physics provides:

- `Raycast`.
- `SphereCast`.
- `CapsuleSweep`.
- `Overlap`.
- Ground query.
- Contact resolution.

The player controller is a swept capsule with:

- Gravity.
- Ground detection.
- Step-up.
- Slope limit.
- Ground snap.
- Air control.
- Jump and pogo bounce.

Simulation runs at a fixed 120 Hz. Rendering interpolates transforms between
simulation states.

## Gameplay

Gameplay systems execute in a deterministic order each fixed tick:

1. Input snapshot.
2. Player controller.
3. Enemy AI.
4. Velocity integration.
5. Physics resolution.
6. Trigger and pickup detection.
7. Damage and death.
8. Animation.
9. Camera.
10. Audio/UI events.
11. Render extraction.

Game states are managed by `engine/scenes`:

- Boot.
- Title.
- Options.
- Controls.
- Map.
- Playing.
- Paused.
- Dying.
- Level complete.
- Game over.

## Audio

SDL supplies the audio device. Go generates/queues PCM through a mixer:

- Master volume.
- Listener position and orientation.
- Simple distance attenuation and panning.
- Procedural sound effects generated at startup.
- Future WAV clips loaded from the asset pack.

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

Validates level JSON against the asset manifest and entity registry.

### gokeen-assetview

Development viewer for levels, meshes, materials, skeletons, and animations.
It is intended to become the foundation of the future level editor viewport.

## Testing

- Unit tests live next to packages.
- Testdata is small and checked in.
- Legacy parser tests use both synthetic chunks and selected real assets.
- Import verification uses structural invariants rather than pixel-perfect
  comparisons.
- Vulkan smoke tests must skip cleanly if no usable device exists.
- Every milestone ends with `gofmt`, `go vet`, and `go test ./...`.

## Error Handling

- Parsers validate chunk lengths and offsets before reading.
- Asset loaders return contextual errors with the asset ID and section.
- Level validation reports all errors, not just the first one.
- Runtime asset lookup failures are fatal during level load.
- Unsupported legacy features are reported during conversion rather than ignored.
