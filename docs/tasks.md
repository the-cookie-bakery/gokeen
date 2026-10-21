# GoKeen Task Tracker

This file tracks only outstanding or incomplete work. Completed work is
summarized in `current-state.md` and is intentionally not kept as finished task
rows here.

Status values in the `Verified` column:

- `partial` - implemented, tests pass for the covered part, but acceptance
  criteria are incomplete.
- `no` - not implemented, not tested, or known incomplete.

## Legacy Data and Backport

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M0-02 | Decode `.dat` record layout, axis order, and entity semantics | Unit tests against real `level1.dat`/`level2.dat` plus bounds validation | partial | Header/record size known; exact axis/semantics still not proven. |
| M0-03 | Recover item, exit, and map-entrance placement data | Level JSON validation + manual in-game check | partial | Level 1 exit is B3D-derived; pickups are authored placeholders. |
| M2-07 | Export skinned B3D meshes/clips to glTF | Verification report | partial | Static glTF export works; skins/animations are pack-only. |
| M2-09 | Import `.dat` data into level JSON | Golden tests against known records | partial | Parser exists; no production entity conversion yet. |
| M2-15 | Asset/model/animation viewer | Manual visual verification | partial | `gokeen -level <id> -fly` exists; no dedicated model viewer. |

## Platform and Input

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M1-05 | Gamepad input support | Build + controller smoke test | partial | Keyboard/mouse work; gamepad events/actions pending. |
| M1-06 | SDL2 audio device and queued PCM | Build + audible smoke test | no | Audio is initialised but no device/mixer exists. |
| M1-08 | Swapchain resize hardening | Automated resize test | partial | Resize path exists; no automated recreation test. |

## Entity and Level Structure

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M2-11 | Registry-driven runtime spawning | `go test ./entities/... ./levels` | partial | Registry exists; runtime uses an ad-hoc switch for Yorp/pickup/exit. |
| M6-01 | Level 2 entity placement and exit | Level validation + playthrough | partial | Geometry imported; no entities/exit yet. |
| M6-02 | Level 5 entity/pogo placement | Level validation + playthrough | partial | Geometry imported; no entities/exit yet. |
| M6-03 | Map hub entities and level entrances | Level validation + playthrough | partial | Geometry imported; hub gameplay pending. |
| M6-09 | Advance backport manifest statuses | `gokeen-levelcheck` | partial | Levels remain `imported`; `validated`/`playable` needs acceptance. |

## Rendering and Lighting

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M3-02 | Full lightmap/material-layer validation | Visual comparison on Level 1/5 | partial | Base + lightmap path works; broader layer combinations unverified. |
| M3-03 | Point/spot lights from JSON | Unit test light parsing + visual test | partial | Directional sun + ambient active; point/spot pending. |
| M3-04 | HDR offscreen target | Renderer smoke test | no | Current rendering goes directly to the sRGB swapchain. |
| M3-05 | Tone mapping/gamma composite | Shader compile + visual test | no | Depends on HDR target. |
| M3-06 | Alpha-test/alpha-blend/additive pipelines | Visual test | no | Signs, glass, effects need these. |
| M3-07 | FXAA or MSAA | Visual test | no | Not implemented. |
| M3-08 | Skybox and fog | Visual test | no | Not implemented. |
| M3-09 | Blob shadows | Visual test | no | No character grounding shadow yet. |
| M3-10 | Directional shadow maps | Visual test + performance check | no | Not implemented. |
| M3-11 | Optional SSAO/bloom | Visual test | no | Not implemented. |
| M3-12 | Normal mapping support | Visual test | no | Tangents/normal maps not wired. |

## Player, Physics, Camera, Animation

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M4-02 | Step-up/slope/ledge tuning | Physics/gameplay tests | partial | Basic capsule movement/landing works; tuning pending. |
| M4-04 | Run simulation on fixed 120 Hz tick | Determinism/replay test | partial | Clock exists; runtime still uses variable frame delta. |
| M4-06 | Third-person spring-arm collision | Camera collision tests + visual | partial | Camera orbit works; no obstruction handling. |
| M4-10 | Death animation and game-over flow | Gameplay test | partial | Retargeted `KeenDie` squash/twitch plays for 1.2s, then respawns; no game-over/life-exhaustion state. |
| M4-13 | Animation blending/transitions | Animation tests + visual | partial | Death/pogo clips are wired; state changes still snap with no cross-fade. |
| M4-14 | Refine gun grip on hand bone | Visual check | partial | Gun follows `HandR`; exact grip offset/rotation still tuned manually. |
| M4-15 | Pogo attachment/animation | Importer test + visual check | partial | `KeenPogo.3ds` skinned to labelled skeleton with a `Pogo` bone; bind-pose clip keeps the feet planted; no hand IK yet. |
| M4-16 | Retarget legacy `KeenDie` rig | Importer test + visual check | partial | `joint1..5` mapped to `PelvisC`/ankle/foot bones; body squash + inverse `Head` scale keep the helmet full-size; no authored full-body death clip. |

## Level 1 Vertical Slice

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M5-04 | Impact effects (Zap/Zot) | Visual test | no | Projectiles hit, but no impact sprite/flash. |
| M5-05 | Richer Yorp AI | AI simulation tests | partial | Movement/landing/stun exist; patrol/chase/ledge logic incomplete. |
| M5-06 | Yorp `YorpConk` animation | Visual/animation test | partial | Stun state works; Yorp still uses a static mesh. |
| M5-07 | Yorp death (`YorpDie`) | Gameplay test | no | No death rule/animation yet. |
| M5-08 | Remaining item models/pickups | Visual test | partial | Pogo/gun/key/pizza/soda work; lollipop/book/bear/signs pending. |
| M5-09 | HUD state presentation | Visual test | partial | State exists; only window-title debug HUD is rendered. |
| M5-10 | Real exit placement in all levels | Playthrough test | partial | Level 1 -> Level 2 works; other levels have no exits. |
| M5-11 | HUD/stats art | Visual test | no | Use old GUI assets or new UI. |
| M5-12 | Procedural SFX and mixer | Audible smoke test | no | Laser, pogo, pickup, hit, hurt sounds pending. |
| M5-13 | Level 1 fully completable/tuned | Replay test + playthrough | partial | Loop exists; placement and tuning incomplete. |

## Enemies, UI, and Game Flow

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M6-04 | Garg model/AI/animations | Gameplay test | no | Assets exist; not imported/used. |
| M6-05 | PatPat support | Gameplay test | no | Assets exist; not imported/used. |
| M6-06 | Title and controls screens | Visual test | no | GUI art exists. |
| M6-07 | Options: fullscreen/resolution/audio | Settings tests | no | No settings UI/persistence. |
| M6-08 | Pause and game-over screens | State tests | no | No scene/state manager. |
| M6-10 | Verify fully self-contained runtime build | Build without `old_game` | no | Embedded assets are used, but this is not yet asserted in CI. |

## Polish and Editor Readiness

| ID | Task | Tests | Verified | Notes |
|---|---|---|---|---|
| M7-01 | Performance profiling and culling | Frame timing report | no | No frustum culling/spatial batching yet. |
| M7-02 | Spatial chunking and batching | Renderer stats test | no | Large map still relies on a few large meshes. |
| M7-03 | Save/settings persistence | Unit tests | no | No config/save files. |
| M7-04 | Level JSON round-trip preservation | `go test ./levels` | no | Unknown editor fields are not preserved. |
| M7-05 | Editor schema exposure | `go test ./entities/...` | partial | Registry metadata exists; runtime/editor schema incomplete. |
| M7-06 | Editor viewport foundation | Manual check | no | Only a fly-camera level viewer exists. |
| M7-07 | Reproducible conversion report checks | Hash comparison test | partial | Pack output is deterministic; report verification is not automated. |
| M7-08 | Windows/macOS build support | Cross-platform build check | no | Linux-first cgo/SDL/Vulkan implementation only. |

## Latest Verification Snapshot

```text
go test -count=1 ./...   PASS
go vet ./...             PASS
make assetcheck          PASS
make levelcheck          PASS

gokeen -frames 3                 PASS
gokeen -spawn-at-exit            Level 1 -> Level 2 PASS
gokeen -autofire                 projectile/gun visual check PASS
gokeen + hold W                  skinned walk animation visual check PASS
gokeen -pogo                     pogo attachment/toggle visual check PASS
gokeen -die                      retargeted death/squash visual check PASS
gokeen + press K                 death pose + 1.2s respawn check PASS
ESC / SIGTERM exit               cursor-release smoke check PASS
```

Automated renderer screenshot tests are still missing; visual checks are manual.
