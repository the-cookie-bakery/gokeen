//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gokeen/assets"
	"gokeen/engine/animations"
	"gokeen/engine/core"
	maths "gokeen/engine/mathematics"
	"gokeen/engine/physics"
	"gokeen/engine/platforms/sdl2"
	"gokeen/engine/renderers/vulkan"
	"gokeen/gameplay/cameras"
	"gokeen/gameplay/enemies"
	"gokeen/gameplay/pickups"
	"gokeen/gameplay/players"
	"gokeen/gameplay/triggers"
	"gokeen/gameplay/weapons"
	"gokeen/levels"
)

const playerModelScale = float32(0.036)

type config struct {
	width       int
	height      int
	fullscreen  bool
	frames      int
	title       string
	fly         bool
	pogo        bool
	autofire    bool
	spawnAtExit bool
	die         bool
}

type gameRuntime struct {
	window     *sdl2.Window
	renderer   *vulkan.Renderer
	store      *assets.Store
	pack       *assets.Pack
	cfg        config
	keys       map[int]bool
	mouse      map[int]bool
	clock      *core.WallClock
	frameCount int
	player     *players.Controller
	signals    chan os.Signal
}

type activeEnemy struct {
	enemy    *enemies.Enemy
	instance *vulkan.MeshInstance
}

type activePickup struct {
	cfg       pickups.Config
	mesh      assets.Mesh
	instance  *vulkan.MeshInstance
	position  maths.Vec3
	offset    maths.Vec3
	scale     float32
	phase     float32
	collected bool
}

type activeExit struct {
	target   string
	position maths.Vec3
}

func main() {
	width := flag.Int("width", 1280, "window width")
	height := flag.Int("height", 720, "window height")
	fullscreen := flag.Bool("fullscreen", false, "start fullscreen")
	frames := flag.Int("frames", 0, "run this many frames then exit (0 = forever)")
	title := flag.String("title", "GoKeen", "window title")
	levelID := flag.String("level", "level1", "embedded level id")
	fly := flag.Bool("fly", false, "use the free-fly viewer camera instead of the player")
	pogo := flag.Bool("pogo", false, "start with the pogo stick equipped and deployed")
	autofire := flag.Bool("autofire", false, "hold the laser trigger (debug)")
	spawnAtExit := flag.Bool("spawn-at-exit", false, "spawn on the first exit trigger (debug)")
	die := flag.Bool("die", false, "show the legacy death model (debug)")
	flag.Parse()

	cfg := config{
		width: *width, height: *height, fullscreen: *fullscreen,
		frames: *frames, title: *title, fly: *fly, pogo: *pogo, autofire: *autofire,
		spawnAtExit: *spawnAtExit, die: *die,
	}
	if err := run(cfg, levels.LevelID(*levelID)); err != nil {
		fmt.Fprintln(os.Stderr, "gokeen:", err)
		os.Exit(1)
	}
}

func run(cfg config, start levels.LevelID) error {
	pack, err := assets.LoadEmbedded()
	if err != nil {
		return fmt.Errorf("load embedded assets: %w", err)
	}
	store, err := assets.NewStore(pack)
	if err != nil {
		return fmt.Errorf("validate embedded assets: %w", err)
	}
	if err := sdl2.Init(); err != nil {
		return err
	}
	defer sdl2.Quit()
	window, err := sdl2.CreateWindow(cfg.title, cfg.width, cfg.height, cfg.fullscreen)
	if err != nil {
		return err
	}
	defer window.Destroy()
	defer window.ReleaseMouse()
	window.CaptureMouse()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	renderer, err := vulkan.New(window, cfg.width, cfg.height)
	if err != nil {
		return err
	}
	defer renderer.Destroy()

	player := players.New(maths.Vec3{})
	player.PogoEnabled = cfg.pogo
	player.PogoDeployed = cfg.pogo
	player.HasGun = true
	player.Ammo = 50
	rt := &gameRuntime{
		window: window, renderer: renderer, store: store, pack: pack, cfg: cfg,
		keys: make(map[int]bool), mouse: make(map[int]bool),
		clock: core.NewWallClock(), player: player, signals: signals,
	}
	current := start
	for {
		next, quit, err := rt.playLevel(current)
		if err != nil {
			return err
		}
		if quit || next == "" {
			return nil
		}
		current = levels.LevelID(next)
	}
}

func (g *gameRuntime) playLevel(id levels.LevelID) (next string, quit bool, err error) {
	level, err := levels.LoadEmbedded(id, levels.StoreResolver{Store: g.store}, nil)
	if err != nil {
		return "", false, fmt.Errorf("load embedded level: %w", err)
	}
	fmt.Printf("GoKeen: level %s loaded (%d entities)\n", level.ID, len(level.Entities))
	if err := g.renderer.UploadScene(g.store, level); err != nil {
		return "", false, fmt.Errorf("upload scene: %w", err)
	}
	collision := g.buildCollision(level)
	spawn := level.Spawn.Position.Vector()
	g.player.Spawn = spawn
	g.player.Position = spawn
	g.player.Velocity = maths.Vec3{}
	g.player.ModelYawOffset = 0
	g.player.Grounded = false

	flyCam := camera{
		position: spawn.Add(maths.V3(0, 3, 12)),
		yaw:      0,
		pitch:    -0.15,
		speed:    12,
	}
	third := cameras.NewThirdPerson(g.player.Center())
	var playerInstance *vulkan.SkinnedInstance
	var playerPogoInstance *vulkan.SkinnedInstance
	var gunInstance *vulkan.MeshInstance
	var playerModelOffsetY, pogoModelOffsetY, pogoRise, playerMinY, deathBoneDrop float32
	var enemiesList []activeEnemy
	var pickupList []activePickup
	var exitList []activeExit
	var laserCooldown, shootTimer float32
	type projectileSlot struct {
		projectile *weapons.Projectile
		instance   *vulkan.MeshInstance
	}
	projectiles := make([]projectileSlot, 16)

	if !g.cfg.fly {
		playerInstance, err = g.renderer.AddSkinnedInstance(g.store, "player_keen_skinned", map[string]animations.ClipID{
			"walk":  "clip_keen_walk",
			"shoot": "clip_keen_shoot",
			"jump":  "clip_keen_jump",
			"die":   "clip_keen_die",
		})
		if err != nil {
			return "", false, fmt.Errorf("add skinned player mesh: %w", err)
		}
		playerInstance.SetClip("walk")
		playerInstance.SetPaused(true)
		playerPogoInstance, err = g.renderer.AddSkinnedInstance(g.store, "player_keen_pogo", map[string]animations.ClipID{
			"pogo": "clip_keen_pogo",
		})
		if err != nil {
			return "", false, fmt.Errorf("add pogo player mesh: %w", err)
		}
		playerPogoInstance.SetClip("pogo")
		playerPogoInstance.SetTransform(maths.M4Scale(maths.Vec3{}))
		playerMesh, err := g.store.Mesh("player_keen_skinned")
		if err != nil {
			return "", false, err
		}
		pogoMesh, err := g.store.Mesh("player_keen_pogo")
		if err != nil {
			return "", false, err
		}
		// The pogo mesh includes the stick below the feet. Offset it by its own
		// bounds so the stick touches the ground, then raise the camera target by
		// the extra height so Keen remains framed while standing on it.
		playerMinY = playerMesh.Bounds.Min.Y
		playerModelOffsetY = -playerMinY * playerModelScale
		pogoModelOffsetY = -pogoMesh.Bounds.Min.Y * playerModelScale
		pogoRise = pogoModelOffsetY - playerModelOffsetY
		// The death clip squashes everything below ShoulderC. Compute how far
		// that lifts the feet so the model can be lowered to keep them grounded.
		skeleton, err := g.store.Skeleton(playerMesh.Skeleton)
		if err != nil {
			return "", false, err
		}
		local := make([]maths.Transform, len(skeleton.Bones))
		for i, bone := range skeleton.Bones {
			local[i] = bone.Local
		}
		worlds := animations.BuildPose(&skeleton, local).World
		for i, bone := range skeleton.Bones {
			if bone.Name == "ShoulderC" {
				deathBoneDrop = (worlds[i].TransformPoint(maths.Vec3{}).Y - playerMinY) * playerModelScale
				break
			}
		}
		gunInstance, err = g.renderer.AddMeshInstance(g.store, "item_gun")
		if err != nil {
			return "", false, fmt.Errorf("add gun mesh: %w", err)
		}
		for i := range projectiles {
			inst, err := g.renderer.AddMeshInstance(g.store, "laser_bolt")
			if err != nil {
				return "", false, fmt.Errorf("add projectile mesh: %w", err)
			}
			inst.SetTransform(maths.M4Scale(maths.Vec3{}))
			projectiles[i] = projectileSlot{instance: inst}
		}
		for _, spec := range level.Entities {
			switch spec.Type {
			case "yorp":
				enemy := enemies.New(spec.Transform.Position.Vector())
				instance, err := g.renderer.AddMeshInstance(g.store, "yorp")
				if err != nil {
					return "", false, fmt.Errorf("add yorp mesh: %w", err)
				}
				enemiesList = append(enemiesList, activeEnemy{enemy: enemy, instance: instance})
			case "pickup":
				var cfg pickups.Config
				if len(spec.Properties) > 0 {
					if err := json.Unmarshal(spec.Properties, &cfg); err != nil {
						return "", false, fmt.Errorf("pickup %s: %w", spec.ID, err)
					}
				}
				meshID := pickupMeshID(cfg.Item)
				if meshID == "" {
					fmt.Printf("warning: pickup %s has unsupported item %q\n", spec.ID, cfg.Item)
					continue
				}
				mesh, err := g.store.Mesh(meshID)
				if err != nil {
					return "", false, err
				}
				instance, err := g.renderer.AddMeshInstance(g.store, meshID)
				if err != nil {
					return "", false, fmt.Errorf("add pickup mesh %s: %w", meshID, err)
				}
				scale := float32(0.8)
				offset := maths.V3(0, -mesh.Bounds.Min.Y*scale, 0)
				pickupList = append(pickupList, activePickup{
					cfg: cfg, mesh: mesh, instance: instance,
					position: spec.Transform.Position.Vector(), offset: offset, scale: scale,
					phase: float32(len(pickupList)) * 1.7,
				})
				_ = mesh
			case "exit":
				var cfg struct {
					TargetLevel string `json:"target_level"`
				}
				if len(spec.Properties) > 0 {
					_ = json.Unmarshal(spec.Properties, &cfg)
				}
				mesh, err := g.store.Mesh("exit_sign")
				if err == nil {
					instance, err := g.renderer.AddMeshInstance(g.store, "exit_sign")
					if err != nil {
						return "", false, fmt.Errorf("add exit mesh: %w", err)
					}
					scale := float32(1.0)
					offset := maths.V3(0, -mesh.Bounds.Min.Y*scale, 0)
					pos := spec.Transform.Position.Vector()
					instance.SetTransform(maths.M4Translate(pos.Add(offset)).Mul(maths.M4Scale(maths.V3(scale, scale, scale))))
				}
				exitList = append(exitList, activeExit{target: cfg.TargetLevel, position: spec.Transform.Position.Vector()})
			}
		}
	}

	if g.cfg.spawnAtExit && len(exitList) > 0 {
		g.player.Position = exitList[0].position
		g.player.Velocity = maths.Vec3{}
		fmt.Println("debug: spawning at exit")
	}
	animTime := float32(0)
	deathStarted := false
	deathActive := false
	deathElapsed := float32(0)
	deathTimer := float32(0)
	running := true
	for running {
		select {
		case <-g.signals:
			g.window.ReleaseMouse()
			return "", true, nil
		default:
		}
		for {
			event, ok := g.window.Poll()
			if !ok {
				break
			}
			switch event.Type {
			case sdl2.EventQuit:
				g.window.ReleaseMouse()
				return "", true, nil
			case sdl2.EventKey:
				wasDown := g.keys[event.Key]
				g.keys[event.Key] = event.Action == 1
				if event.Action == 1 && !wasDown && (event.Key == 1073742048 || event.Key == 1073742050) {
					g.player.TogglePogo()
				}
				if event.Action == 1 && !wasDown && event.Key == int('k') {
					g.player.Health = 0
				}
				if event.Action == 1 && event.Key == 27 {
					g.window.ReleaseMouse()
					return "", true, nil
				}
			case sdl2.EventMouseButton:
				g.mouse[event.Button] = event.Action == 1
			case sdl2.EventMouseMove:
				if g.cfg.fly {
					flyCam.yaw += float32(event.MouseDX) * 0.002
					flyCam.pitch -= float32(event.MouseDY) * 0.002
					flyCam.pitch = clamp(flyCam.pitch, -1.4, 1.4)
				} else {
					third.Look(float32(event.MouseDX), float32(event.MouseDY))
				}
			case sdl2.EventResize:
				if event.Width > 0 && event.Height > 0 {
					if err := g.renderer.Resize(event.Width, event.Height); err != nil {
						return "", false, err
					}
				}
			}
		}
		dt := float32(g.clock.Delta())
		if dt > 0.1 {
			dt = 0.1
		}
		animTime += dt
		w, h := g.renderer.Extent()
		if g.cfg.fly {
			flyCam.update(dt, g.keys)
			if w > 0 && h > 0 {
				flyCam.setCamera(g.renderer, float32(w)/float32(h))
			}
		} else {
			input := players.Input{Yaw: third.Yaw}
			if g.keys[int('w')] || g.keys[1073741906] {
				input.Forward = 1
			}
			if g.keys[int('s')] || g.keys[1073741905] {
				input.Forward = -1
			}
			if g.keys[int('a')] || g.keys[1073741904] {
				input.Strafe = -1
			}
			if g.keys[int('d')] || g.keys[1073741903] {
				input.Strafe = 1
			}
			input.Jump = g.keys[int(' ')]
			if !deathActive && (g.cfg.die || g.player.Health <= 0) {
				deathActive = true
				deathStarted = false
				deathElapsed = 0
				deathTimer = 1.2
			}
			if deathActive {
				g.player.Update(dt, players.Input{}, collision)
			} else {
				g.player.Update(dt, input, collision)
			}
			aim := third.Forward()
			up := maths.V3(0, 1, 0)
			right := aim.Cross(up).Normalize()
			pogoActive := !deathActive && g.player.PogoEnabled && g.player.PogoDeployed
			if shootTimer > 0 {
				shootTimer -= dt
			}
			var gunPos maths.Vec3
			if deathActive {
				const deathSquashAmount = float32(0.30)
				deathElapsed += dt
				squashT := clamp(deathElapsed/0.15, 0, 1)
				deathSquash := float32(1) + (deathSquashAmount-1)*squashT
				if !deathStarted {
					playerInstance.SetClip("die")
					playerInstance.SetPaused(false)
					deathStarted = true
				}
				playerInstance.Update(dt)
				deathDrop := (1 - deathSquash) * deathBoneDrop
				deathOffsetY := playerModelOffsetY - deathDrop
				deathModel := g.player.ModelMatrix(maths.V3(0, deathOffsetY, 0), playerModelScale)
				playerInstance.SetTransform(deathModel)
				playerPogoInstance.SetTransform(maths.M4Scale(maths.Vec3{}))
				gunInstance.SetTransform(maths.M4Scale(maths.Vec3{}))
				gunPos = g.player.Center()
				if !g.cfg.die {
					deathTimer -= dt
					if deathTimer <= 0 {
						g.player.Respawn()
						g.player.Health = g.player.MaxHealth
						deathActive = false
						deathStarted = false
						deathElapsed = 0
					}
				}
			} else if pogoActive {
				playerPogoInstance.SetClip("pogo")
				playerPogoInstance.SetPaused(true)
				playerPogoInstance.SetHeadPitch(clamp(-third.Pitch, -0.7, 0.7))
				playerPogoInstance.Update(dt)
				pogoModel := g.player.ModelMatrix(maths.V3(0, pogoModelOffsetY, 0), playerModelScale)
				playerPogoInstance.SetTransform(pogoModel)
				playerInstance.SetTransform(maths.M4Scale(maths.Vec3{}))
				gunInstance.SetTransform(maths.M4Scale(maths.Vec3{}))
				gunPos = g.player.Position.Add(maths.V3(0, g.player.Height*0.55+pogoRise, 0)).
					Add(right.Scale(0.5)).Add(aim.Scale(0.3))
			} else {
				moving := input.Forward != 0 || input.Strafe != 0
				if shootTimer > 0 {
					playerInstance.SetClip("shoot")
					playerInstance.SetPaused(false)
				} else if !g.player.Grounded {
					playerInstance.SetClip("jump")
					playerInstance.SetPaused(false)
				} else if moving {
					playerInstance.SetClip("walk")
					playerInstance.SetPaused(false)
				} else {
					playerInstance.SetClip("walk")
					playerInstance.SetPaused(true)
				}
				playerInstance.SetHeadPitch(clamp(-third.Pitch, -0.7, 0.7))
				playerInstance.Update(dt)
				playerModel := g.player.ModelMatrix(maths.V3(0, playerModelOffsetY, 0), playerModelScale)
				playerInstance.SetTransform(playerModel)
				playerPogoInstance.SetTransform(maths.M4Scale(maths.Vec3{}))
				gunPos = g.player.Position.Add(maths.V3(0, g.player.Height*0.55, 0)).
					Add(right.Scale(0.5)).Add(aim.Scale(0.3))
				if handPos, ok := playerInstance.BoneModelPosition("HandR"); ok {
					gunPos = playerModel.TransformPoint(handPos).Add(aim.Scale(0.18)).Add(right.Scale(-0.08))
				}
				gunModel := maths.M4Translate(gunPos).
					Mul(maths.M4Rotation(quatFromForward(aim))).
					Mul(maths.M4Scale(maths.V3(0.8, 0.8, 0.8)))
				gunInstance.SetTransform(gunModel)
			}
			muzzle := gunPos.Add(aim.Scale(0.5))
			shotDir := aim
			for _, active := range enemiesList {
				active.enemy.Update(dt, collision)
				active.instance.SetTransform(active.enemy.ModelMatrix(maths.V3(0, 0.331, 0), 0.7))
			}
			for i := range pickupList {
				p := &pickupList[i]
				if p.collected {
					continue
				}
				bob := float32(math.Sin(float64(animTime*2 + p.phase)))
				pos := p.position.Add(maths.V3(0, bob*0.2, 0))
				model := maths.M4Translate(pos.Add(p.offset)).
					Mul(maths.M4Rotation(maths.QFromAxisAngle(maths.V3(0, 1, 0), animTime*2+p.phase))).
					Mul(maths.M4Scale(maths.V3(p.scale, p.scale, p.scale)))
				p.instance.SetTransform(model)
				if g.player.Center().Distance(pos) < 2.4 {
					message, ok := pickups.Apply(g.player, p.cfg)
					if ok {
						fmt.Printf("picked up %s (score %d, ammo %d, pogo %v, key %v)\n", message, g.player.Score, g.player.Ammo, g.player.PogoEnabled, g.player.BlueKey)
						p.collected = true
						p.instance.SetTransform(maths.M4Scale(maths.Vec3{}))
					}
				}
			}
			if laserCooldown > 0 {
				laserCooldown -= dt
			}
			for i := range projectiles {
				slot := &projectiles[i]
				if slot.projectile == nil || !slot.projectile.Active {
					continue
				}
				previous, current := slot.projectile.Advance(dt)
				delta := current.Sub(previous)
				distance := delta.Length()
				if distance <= 1e-5 || !slot.projectile.Active {
					slot.projectile.Deactivate()
					slot.instance.SetTransform(maths.M4Scale(maths.Vec3{}))
					continue
				}
				dir := delta.Scale(1 / distance)
				hitDist := distance
				hit := false
				if collision != nil {
					if h, ok := collision.Raycast(maths.NewRay(previous, dir), distance); ok {
						hitDist = h.Distance
						hit = true
					}
				}
				for _, active := range enemiesList {
					if d, ok := active.enemy.RayHit(previous, dir, hitDist); ok {
						active.enemy.Stun(2)
						hitDist = d
						hit = true
					}
				}
				if hit {
					slot.projectile.Deactivate()
					slot.instance.SetTransform(maths.M4Scale(maths.Vec3{}))
					continue
				}
				slot.instance.SetTransform(beamMatrix(current.Sub(dir.Scale(0.75)), current.Add(dir.Scale(0.75)), 0.2))
			}
			trigger := !deathActive && (g.keys[int('f')] || g.mouse[1] || g.cfg.autofire)
			if trigger && laserCooldown <= 0 && g.player.HasGun && g.player.Ammo > 0 {
				var slot *projectileSlot
				for i := range projectiles {
					if projectiles[i].projectile == nil || !projectiles[i].projectile.Active {
						slot = &projectiles[i]
						break
					}
				}
				if slot != nil {
					g.player.Ammo--
					slot.projectile = weapons.Spawn(muzzle, shotDir, 30, 3)
					slot.instance.SetTransform(beamMatrix(muzzle.Sub(shotDir.Scale(0.5)), muzzle.Add(shotDir.Scale(0.5)), 0.15))
					laserCooldown = 0.18
					shootTimer = 0.3
				}
			}
			third.Target = g.player.Center()
			if pogoActive {
				third.Target = third.Target.Add(maths.V3(0, pogoRise, 0))
			}
			if w > 0 && h > 0 {
				vp, pos := third.ViewProjection(float32(w) / float32(h))
				g.renderer.SetCamera(vp, pos)
			}
			for _, exit := range exitList {
				if triggers.Reached(g.player.Position, exit.position, 3.2, 6) {
					fmt.Printf("level %s complete", level.ID)
					if exit.target != "" {
						fmt.Printf(" -> %s", exit.target)
					}
					fmt.Println()
					if exit.target == "" {
						return "", true, nil
					}
					return exit.target, false, nil
				}
			}
		}
		state := ""
		if deathActive {
			state = " | DEAD"
		}
		g.window.SetTitle(fmt.Sprintf("GoKeen | %s | HP %d | Ammo %d | Score %d | Lives %d | Pogo %v%s",
			level.ID, g.player.Health, g.player.Ammo, g.player.Score, g.player.Lives, g.player.PogoDeployed, state))
		if err := g.renderer.Frame(); err != nil {
			if errors.Is(err, vulkan.ErrOutOfDate) {
				w, h := g.window.Size()
				if err := g.renderer.Resize(w, h); err != nil {
					return "", false, err
				}
			} else {
				return "", false, err
			}
		}
		g.frameCount++
		if g.cfg.frames > 0 && g.frameCount >= g.cfg.frames {
			return "", true, nil
		}
		time.Sleep(time.Millisecond)
	}
	return "", true, nil
}

func (g *gameRuntime) buildCollision(level *levels.Level) *physics.Mesh {
	if len(level.Geometry) == 0 {
		return nil
	}
	collisionID := level.Geometry[0].Collision
	if collisionID == "" {
		collisionID = level.Geometry[0].Mesh
	}
	mesh, err := g.store.Mesh(assets.MeshID(collisionID))
	if err != nil {
		fmt.Printf("warning: collision mesh %q unavailable: %v\n", collisionID, err)
		return nil
	}
	positions := make([]maths.Vec3, len(mesh.Vertices))
	for i, v := range mesh.Vertices {
		positions[i] = v.Position
	}
	return physics.NewMesh(physics.TrianglesFromIndexed(positions, mesh.Indices))
}

func pickupMeshID(item pickups.Item) assets.MeshID {
	switch item {
	case pickups.Pogo:
		return "item_pogo"
	case pickups.Gun:
		return "item_gun"
	case pickups.BlueKey:
		return "item_key"
	case pickups.Pizza:
		return "item_pizza"
	case pickups.Soda:
		return "item_soda"
	default:
		return ""
	}
}

func quatFromForward(dir maths.Vec3) maths.Quat {
	dir = dir.Normalize()
	from := maths.V3(0, 0, -1)
	dot := from.Dot(dir)
	if dot < -0.9999 {
		return maths.QFromAxisAngle(maths.V3(0, 1, 0), maths.Pi)
	}
	axis := from.Cross(dir).Normalize()
	angle := float32(math.Acos(float64(dot)))
	return maths.QFromAxisAngle(axis, angle)
}

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

type camera struct {
	position maths.Vec3
	yaw      float32
	pitch    float32
	speed    float32
}

func (c *camera) forward() maths.Vec3 {
	cy := float32(math.Cos(float64(c.yaw)))
	sy := float32(math.Sin(float64(c.yaw)))
	cp := float32(math.Cos(float64(c.pitch)))
	sp := float32(math.Sin(float64(c.pitch)))
	return maths.V3(sy*cp, sp, -cy*cp).Normalize()
}

func (c *camera) right() maths.Vec3 {
	return c.forward().Cross(maths.V3(0, 1, 0)).Normalize()
}

func (c *camera) update(dt float32, keys map[int]bool) {
	speed := c.speed
	var move maths.Vec3
	fwd := maths.V3(c.forward().X, 0, c.forward().Z).Normalize()
	right := maths.V3(c.right().X, 0, c.right().Z).Normalize()
	if keys[int('w')] || keys[1073741906] {
		move = move.Add(fwd)
	}
	if keys[int('s')] || keys[1073741905] {
		move = move.Sub(fwd)
	}
	if keys[int('a')] || keys[1073741904] {
		move = move.Sub(right)
	}
	if keys[int('d')] || keys[1073741903] {
		move = move.Add(right)
	}
	if keys[int('q')] {
		move = move.Sub(maths.V3(0, 1, 0))
	}
	if keys[int('e')] {
		move = move.Add(maths.V3(0, 1, 0))
	}
	if move.LengthSquared() > 0 {
		c.position = c.position.Add(move.Normalize().Scale(speed * dt))
	}
	if keys[int('j')] {
		c.yaw += 1.5 * dt
	}
	if keys[int('l')] {
		c.yaw -= 1.5 * dt
	}
	if keys[int('i')] {
		c.pitch += 1.0 * dt
	}
	if keys[int('k')] {
		c.pitch -= 1.0 * dt
	}
	c.pitch = clamp(c.pitch, -1.4, 1.4)
}

func (c *camera) setCamera(renderer *vulkan.Renderer, aspect float32) {
	view := maths.LookAtRH(c.position, c.position.Add(c.forward()), maths.V3(0, 1, 0))
	proj := maths.PerspectiveRH(70*maths.DegToRad, aspect, 0.1, 1000)
	renderer.SetCamera(proj.Mul(view), c.position)
}

func beamMatrix(start, end maths.Vec3, thickness float32) maths.Mat4 {
	d := end.Sub(start)
	length := d.Length()
	if length < 1e-4 {
		return maths.M4Scale(maths.Vec3{})
	}
	z := d.Scale(1 / length)
	up := maths.V3(0, 1, 0)
	if z.Y > 0.99 || z.Y < -0.99 {
		up = maths.V3(1, 0, 0)
	}
	x := up.Cross(z).Normalize()
	y := z.Cross(x).Normalize()
	c := start.Add(end).Scale(0.5)
	return maths.Mat4{
		x.X * thickness, x.Y * thickness, x.Z * thickness, 0,
		y.X * thickness, y.Y * thickness, y.Z * thickness, 0,
		z.X * length, z.Y * length, z.Z * length, 0,
		c.X, c.Y, c.Z, 1,
	}
}
