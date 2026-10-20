# GoKeen Implementation Plan

## Purpose

GoKeen is a new implementation of the unfinished Commander Keen 3D prototype
originally written in Blitz3D. The new game is written in Go, uses SDL2 for
window/input/audio platform integration, and renders through Vulkan with custom
shaders.

The project reuses the original game's art and level content by converting the
legacy `.b3d`, `.3ds`, and `.dat` files into new, editable source assets. The
legacy formats are not used at runtime.

## Locked Decisions

| Area | Decision |
|---|---|
| Language | Go, standard library only, plus `golang.org/x/image` where needed |
| Platform | Linux first; platform code isolated for later Windows/macOS support |
| Window/input/audio | SDL2 through a hand-written cgo bridge |
| Rendering | Vulkan through a hand-written cgo bridge |
| Asset conversion | Offline only; legacy formats never loaded by the game at runtime |
| Editable meshes | glTF 2.0 plus binary buffers and PNG textures |
| Editable levels | JSON with stable IDs and asset references |
| Runtime assets | Generated `.gkp` pack embedded with `go:embed` |
| Lighting | Modern Blinn-Phong, baked lightmaps, dynamic lights, HDR/tone mapping |
| Physics | Custom triangle BVH plus swept capsule character controller |
| Tick rate | 120 Hz fixed simulation; interpolated rendering |
| First milestone target | Level 1 vertical slice |
| Editor readiness | Stable IDs, schemas, round-trippable JSON, dev asset viewer |

## Non-Goals

- Runtime compatibility with Blitz3D data structures.
- Running the original executable or emulating Blitz3D.
- Fabricating missing Levels 3, 4, and 16.
- Building a full level editor during the vertical-slice phase.
- Third-party Go modules for SDL, Vulkan, ECS, or rendering.

## Asset Pipeline

```text
old_game legacy files
        |
        v
gokeen-assetconv
        |
        +--> source/assets/sources/meshes/*.gltf + *.bin
        +--> source/assets/sources/textures/*.png
        +--> source/assets/sources/manifest.json
        +--> source/levels/data/*.json
        +--> build/verification-report.json
        |
        v
packer
        |
        v
source/assets/pack/gokeen.gkp
        |
        v
go:embed -> final binary
```

### Legacy Import Rules

- B3D is parsed as little-endian chunked data.
- Materials are resolved per `TRIS` brush, not only from the `MESH` master brush.
- Two UV sets are supported. Texture flag `0x10000` selects the second UV set.
- Texture blend `5` is treated as multiply (legacy lightmap blending).
- Skeletal nodes, bones, vertex weights, and keyframes become glTF skins and
  animations.
- Static level geometry is flattened to world space and split for culling.
- 3DS files are parsed as big-endian chunked data.
- Legacy texture paths are resolved by relative path first, then by lowercase
  basename, then by deterministic fallback.
- Missing textures produce checker/average-color fallbacks and a report entry.
- Level `.dat` files are decoded in a dedicated Phase 0 task and validated
  against level bounds and walkable surfaces.
- `map2b.b3d` is the hub level.
- `Level1.b3d` and `level1.dat` are the Level 1 sources.
- `Level2ba.b3d` and `level2.dat` are the Level 2 sources.
- `Level5a_gls.b3d` is the Level 5 source; entity placement is authored manually.
- Levels 3, 4, and 16 are tracked as missing assets.

## Level JSON

Levels are the source of truth for entity placement, lighting, triggers, and
geometry instances. The format is versioned and validated strictly.

Core rules:

- Stable string IDs for levels, geometry instances, entities, lights, and triggers.
- Positions and rotations are fully 3D.
- Rotations are Euler degrees in source JSON.
- Geometry references mesh assets; polygon data is not embedded in level JSON.
- No derived runtime data is persisted.
- Optional `editor` metadata supports future layers, groups, locks, and notes.
- Unknown extension data can be preserved for tools that do not understand it.

Entity definitions are registered with a typed schema so the runtime, validator,
and future editor agree on property fields.

## Rendering

- Vulkan forward/forward+ renderer.
- Per-pixel Blinn-Phong shading.
- Ambient, diffuse, specular, emissive, and lightmap contributions.
- Directional, point, and spot lights.
- sRGB texture sampling and linear-space lighting.
- HDR offscreen target, exposure, tone mapping, gamma conversion.
- FXAA baseline; optional MSAA.
- Alpha-test, alpha-blend, and additive pipelines.
- Skeletal skinning with bone palettes.
- Sprite and billboard rendering for lasers, impacts, and shadows.
- Skybox and fog.
- Frustum culling and spatial mesh chunks.
- Blob shadows initially; directional shadow maps in polish.

## Physics and Gameplay

- Triangle BVH generated offline.
- Swept capsule character controller with slide, ground snap, step-up, and slopes.
- Third-person spring-arm camera with collision.
- Fixed 120 Hz simulation.
- Keen: run, jump, pogo bounce, laser, health, lives, respawn.
- Pogo: equipped item, higher chained bounces.
- Laser: ammo, raycast/projectile, impacts, Yorp stun, repeated-hit death.
- Enemies: Yorp first, then Garg and PatPat.
- Items: pogo, sign, teddy, lollipop, pizza, soda, gun, key, book, bear.
- Exits and map level entrances.
- HUD, title, options, controls, pause, game over.

## Milestones

### M0 - Legacy archaeology and conversion specification

- Decode `.dat` semantics and validate placements.
- Inventory assets and missing references.
- Define glTF mapping, coordinate conversion, and material/animation conversion.
- Seed the backport manifest.
- Define verification criteria.

### M1 - Platform and renderer bring-up

- SDL2 cgo bridge: window, events, input, audio device.
- Vulkan cgo bridge: instance, device, swapchain, clear screen.
- Validation layers and resize handling.
- Runnable game binary.

### M2 - Import pipeline and static level viewer

- B3D, 3DS, texture, and `.dat` importers.
- glTF writer and verifier.
- Pack writer/reader and typed asset store.
- Level JSON loader and validator.
- Asset viewer tool.
- Level 1 static geometry renders with textures and lightmaps.

### M3 - Modern lighting and materials

- Blinn-Phong shaders.
- Dynamic lights from level JSON.
- Lightmap blending.
- HDR, tone mapping, gamma, FXAA.
- Alpha and additive passes.

### M4 - Player, physics, camera

- BVH and capsule collision.
- Fixed timestep.
- Third-person camera.
- Keen movement and skeletal animation.
- Slopes, step-up, jumping, respawn.

### M5 - Level 1 vertical slice

- Pogo.
- Laser and impacts.
- Yorp AI, stun, death.
- Pickups, score, health, ammo, key.
- Exit and level transition.
- HUD and sound effects.
- Level 1 completable.

### M6 - Full backport pass

- Level 2, Level 5, and map hub.
- Garg/PatPat.
- Title, options, controls, pause, game over.
- Fullscreen/widescreen.
- Backport manifest updated to validated/playable.

### M7 - Polish and editor readiness

- Performance profiling and culling.
- Directional shadows, optional SSAO/bloom.
- Normal mapping where metadata exists.
- Save/settings persistence.
- JSON round-trip and editor-schema tests.
- Reproducible verification reports.

## Verification Strategy

- Pure Go unit tests for math, ECS, levels, assets, physics, animation, and
  legacy parsers.
- Round-trip tests for asset packs and level JSON.
- Structural verification of imported glTF against legacy import reports:
  mesh counts, triangle counts, vertex counts, bounds, UVs, materials,
  animation tracks, and texture dimensions.
- Deterministic gameplay replay tests for movement, pogo, and raycasts.
- Renderer smoke tests with Vulkan validation layers where possible.
- `gofmt`, `go vet`, and `go test ./...` before each milestone is considered
  verified.

## Backport Tracking

`source/levels/backport.json` is the machine-readable level backlog. Statuses:

- `missing-source`
- `imported`
- `validated`
- `playable`
- `polished`

Initial backlog:

| Level | Source | Entity data | Initial status |
|---|---|---|---|
| map | `Map/map2b.b3d` | manual | imported |
| level1 | `Levels/Level1/Level1.b3d` | `level1.dat` | imported |
| level2 | `Levels/Level2/Level2ba.b3d` | `level2.dat` | imported |
| level5 | `Levels/Level5/Level5a_gls.b3d` | manual | imported |
| level3 | missing | missing | missing-source |
| level4 | missing | missing | missing-source |
| level16 | missing | missing | missing-source |


## Current Implementation Status

The first implementation pass is complete for:

- Core math, fixed-step clock, ECS, BVH/capsule physics, and animation sampling.
- Typed asset IDs, deterministic asset packs, and embedded pack loading.
- Strict level JSON loading, validation, entity registry, and backport manifest.
- Legacy B3D/3DS/.dat parsing, texture resolution, BMP/PNG conversion.
- Static level conversion for `map`, `level1`, `level2`, and `level5`.
- Editable glTF exports, conversion reports, and structural verification.
- SDL2 window/input bridge, Vulkan instance/device/swapchain, clear/present.
- Runnable `gokeen` executable with embedded assets.
- Vulkan static mesh pipeline with depth buffer, textures, lightmaps, and
  Blinn-Phong shading.
- Fly-camera level viewer via `gokeen -level <id>`.
- Third-person Keen model, capsule movement, jumping, pogo bounce, and a
  basic Yorp actor.
- Hitscan laser with a visible beam and Yorp stun state.

Still pending: skinned animation pipeline, shadow maps, HDR/post-processing,
pickup/exit/HUD integration, audio, full entity placement, and the editor
foundation.
