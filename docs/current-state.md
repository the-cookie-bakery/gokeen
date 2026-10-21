# GoKeen Current State

This document is the short, factual snapshot of the repository. The detailed
backlog remains in `tasks.md`, the roadmap in `implementation-plan.md`, and the
design in `architecture.md`. Format and implementation gotchas are collected in
`lessons-learned.md`.

## Summary

GoKeen is now a runnable Linux prototype with:

- Embedded legacy-derived assets.
- Vulkan-rendered static levels.
- A third-person Keen player with collision, jumping, and pogo bounce.
- Skinned walk/jump/shoot animation for Keen.
- A hand-held gun and moving laser projectiles.
- Yorp actors with a stun state.
- Pickups, player state, exit triggers, and level transitions.

It is not yet a complete game. Animation, HUD art, audio, full level entity
placement, and the remaining levels' gameplay loops are still pending.

## Build and Run

From `source/`:

```sh
make generate      # legacy assets -> glTF/JSON/pack, compile shaders
make test          # go test ./...
make vet           # go vet ./...
make build         # binaries in build/
make run           # run Level 1
```

Direct run examples:

```sh
go run ./cmd/gokeen -level level1
go run ./cmd/gokeen -level map -fly
go run ./cmd/gokeen -level level2 -pogo -autofire
go run ./cmd/gokeen -spawn-at-exit -frames 5   # verify Level 1 -> Level 2
```

## Controls

| Input | Action |
|---|---|
| `W`, `A`, `S`, `D` | Move |
| Mouse | Look |
| `Space` | Jump / pogo bounce while deployed |
| `Ctrl` | Deploy or retract the pogo stick (once owned) |
| `F` or left mouse | Fire laser |
| `Escape` | Quit |

Debug keys in player mode: `K` kills Keen to verify the death/respawn flow.
Debug/viewer keys in fly mode: `WASD` move, `Q`/`E` down/up, `I`/`J`/`K`/`L`
look around.

## Implemented

### Assets and conversion

- Typed asset IDs: mesh, material, texture, skeleton, clip, audio, level.
- Deterministic `.gkp` pack with content hash and section validation.
- Embedded pack loading with `go:embed`.
- B3D and 3DS parsers with real-asset tests.
- Texture path resolver, BMP/PNG conversion, missing-texture fallback.
- Static B3D/3DS conversion to glTF, source manifest, and runtime pack.
- Level `.dat` record parser (semantics still partially unresolved).
- Asset conversion, asset verification, and level validation CLIs.

Current pack:

- 21 meshes
- 117 materials
- 35 textures
- 2 skeletons (Keen 26 bones, legacy death 6 bones)
- 6 animation clips (`walk`, `jump`, `shoot`, `pogo`, `die`, legacy `death`)
- approximately 6.0 MiB

### Rendering

- SDL2 window/surface.
- Vulkan instance, physical/logical device, swapchain.
- Depth buffer and resize/recreation.
- Static mesh pipeline with vertex/index buffers and textures.
- Skinned mesh pipeline with a 64-bone palette.
- Per-material descriptor sets and per-instance bone buffers.
- Blinn-Phong directional/ambient shading.
- Lightmap texture layer using UV1 where available.
- Push-constant material/model data.
- Fly camera and third-person camera with corrected mouse direction.

Not implemented in the renderer yet:

- Alpha-test, alpha-blend, additive, sprite, skybox, UI pipelines.
- HDR target, tone mapping, FXAA, bloom, SSAO.
- Shadow maps.
- Point/spot lights.
- Validation layers.
- Frustum culling and spatial batching.

### Physics and player

- Triangle BVH.
- Raycast and overlap queries.
- Capsule contact resolution and conservative swept movement.
- Fixed-step clock implemented and tested; the runtime loop still uses variable
  frame delta.
- Camera-relative movement, gravity, jumping.
- Pogo bounce.
- Fall respawn and lives counter.
- No tuned step-up/slope/ledge handling yet.
- Third-person camera has no spring-arm collision yet.

### Gameplay

- Skinned Keen model with walk/jump/shoot animation and head pitch.
- Character body rotates with camera yaw (correct left/right direction).
- All skinned vertices are weighted; unweighted vertices bind to the root bone.
- Pogo pickup grants the pogo stick; Ctrl deploys or retracts it.
- `KeenPogo.3ds` is skinned to the labelled Keen skeleton. Helmet/face/skin
  map to `Head`, hands/watch to hand/wrist bones, shoes to feet, jeans/shirt to
  the body chain, and the pogo stick to a dedicated root-child `Pogo` bone.
- The `pogo` clip is bind-pose only, so Keen stands still on the stick; the
  bounce comes from the controller/whole-model movement, not foot animation.
- Pogo bounce uses a higher launch speed and chains while Space is held.
- The legacy `KeenDie` joint deltas are retargeted from `joint1..5` to
  `PelvisC`/`AnkleL`/`AnkleR`/`FootL`/`FootR` on the labelled Keen skeleton.
- Death compresses the body with a `ShoulderC` Y scale while inverse-scaling
  `Head`, so the body squeezes together but the helmet/head keeps its original
  size.
- The normal labelled Keen mesh is used for death rather than the legacy
  head+feet shell.
- `K` triggers the death state; after 1.2s Keen respawns and loses a life.
- Gun pickup adds ammo.
- Hand-held gun model follows the right-hand bone and aim direction.
- Moving projectile laser consumes ammo, has a cooldown, and stuns Yorps on hit.
- Cursor is released and shown again on normal exit, Ctrl-C, or SIGTERM.
- Key, pizza, and soda pickups have state effects.
- Score, health, ammo, lives, key, and pogo state persist across level changes.
- Yorp actors move, fall, land, and can be stunned.
- Exit triggers switch levels.
- Window title acts as a temporary HUD.

Not implemented yet:

- Real HUD/stats art.
- Audio playback and SFX mixer.
- Yorp skinned animation, death, and impact sprites.
- Animation blending/transitions between states.
- Garg, PatPat, and richer enemy AI.
- Game-over screen and life-exhaustion handling.
- Save/options UI.
- Complete entity placement for Levels 2, 5, and the map hub.

### Content status

| Level | Geometry | Entity placement | Runtime status |
|---|---|---|---|
| `map` | imported | none yet | Loads and renders |
| `level1` | imported | 1 Yorp, pogo, gun, exit | Playable loop with transition |
| `level2` | imported | none yet | Loads and renders |
| `level5` | imported | none yet | Loads and renders |
| `level3` | missing | missing | Not backportable |
| `level4` | missing | missing | Not backportable |
| `level16` | missing | missing | Not backportable |

Level 1 currently has:

- A pogo pickup and gun pickup near the spawn.
- One Yorp actor.
- An exit based on the legacy `Exit_cube` position, transitioning to Level 2.

Level 2, Level 5, and the map still need authored entity placement and exits.

## Verification

Current acceptance commands:

```text
go test -count=1 ./...   PASS
go vet ./...             PASS
make assetcheck          PASS
make levelcheck          PASS
go run ./cmd/gokeen -spawn-at-exit -frames 5
                         Level 1 -> Level 2 transition verified
go run ./cmd/gokeen -autofire
                         moving projectile and gun visual check
go run ./cmd/gokeen -level level1 -pogo
                         pogo attachment, Ctrl toggle, and bounce visual check
go run ./cmd/gokeen -level level1 -die
                         legacy death animation/retarget visual check
gokeen + press K          death pose + 1.2s respawn check
```

Renderer smoke checks have been performed manually with screenshots on the
local X11 display. There is no automated pixel test yet.

## Recent Fixes

- Mouse cursor is released and shown on ESC/quit, Ctrl-C, and SIGTERM; mouse grab is removed and an atexit fallback restores the cursor.
- Mouse-right now rotates the camera and character right.
- B3D quaternion convention now matches Blitz3D correctly.
- Unweighted skinned vertices bind to the root bone instead of collapsing.
- Keen now plays walk/jump/shoot clips with front/back leg and arm motion.
- Head pitch follows camera elevation with the correct sign.
- Gun follows the right hand instead of floating beside the character.
- Laser shots are moving projectiles that collide with geometry and enemies.
- `KeenPogo.3ds` is now skinned to the labelled Keen skeleton instead of being
  attached as an unlabelled root mesh.
- Ctrl toggles pogo deployment; Space jumps or pogo-bounces while deployed.
- The pogo mesh follows the player, keeps its bottom on the ground, and keeps
  the authored standing-on-pogo pose.
- The legacy `KeenDie` animation is retargeted onto the labelled skeleton and
  drives a squashed death pose on the normal Keen mesh.
- The body squash keeps the helmet/head at full size via a compensating `Head`
  Y scale.
- `K` triggers death, the death pose holds for 1.2s, then Keen respawns.

## Known Limitations

- The `.dat` entity format is only partially decoded; enemy/item/exit placement
  is currently authored JSON rather than fully recovered from legacy data.
- The renderer has no alpha/sprite/UI passes, so effects and a real HUD are
  missing.
- Audio is initialised through SDL but no device/mixer is created yet.
- Only Keen is animated; Yorps still use a static mesh.
- The pogo mesh uses rigid material-to-bone weights. Its hands stay in the
  legacy pogo pose; the pogo clip is bind-pose only, so the feet do not animate
  and there is no hand IK.
- `KeenDie.b3d` is a squashed head+feet shell with a `ROOT`+`joint1..5` rig.
  Its joint deltas are retargeted onto the labelled Keen skeleton. Body squash
  and head-size compensation are generated scale tracks, not an authored
  full-body death clip.
- Death always respawns after the 1.2s death timer; there is no game-over state
  when lives reach zero.
- The gun follows the right-hand bone; the exact grip offset can still be refined.
- Level 2/5/map are renderable but not yet gameplay-complete.
- The exit trigger uses a forgiving vertical range because the exact legacy
  trigger volume has not been recovered.
- The player currently starts with the gun enabled and 50 ammo for testing,
  even though the gun pickup also exists.
