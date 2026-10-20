//go:build linux

package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"gokeen/assets"
	"gokeen/engine/core"
	"gokeen/engine/mathematics"
	"gokeen/engine/physics"
	"gokeen/engine/platforms/sdl2"
	"gokeen/engine/renderers/vulkan"
	"gokeen/gameplay/cameras"
	"gokeen/gameplay/enemies"
	"gokeen/gameplay/players"
	"gokeen/levels"
)

func main() {
	width := flag.Int("width", 1280, "window width")
	height := flag.Int("height", 720, "window height")
	fullscreen := flag.Bool("fullscreen", false, "start fullscreen")
	frames := flag.Int("frames", 0, "run this many frames then exit (0 = forever)")
	title := flag.String("title", "GoKeen", "window title")
	levelID := flag.String("level", "level1", "embedded level id")
	fly := flag.Bool("fly", false, "use the free-fly viewer camera instead of the player")
	pogo := flag.Bool("pogo", false, "start with the pogo stick enabled")
	autofire := flag.Bool("autofire", false, "hold the laser trigger (debug)")
	flag.Parse()

	if err := run(*width, *height, *fullscreen, *frames, *title, *levelID, *fly, *pogo, *autofire); err != nil {
		fmt.Fprintln(os.Stderr, "gokeen:", err)
		os.Exit(1)
	}
}

func run(width, height int, fullscreen bool, frames int, title, levelID string, fly, pogo, autofire bool) error {
	pack, err := assets.LoadEmbedded()
	if err != nil {
		return fmt.Errorf("load embedded assets: %w", err)
	}
	store, err := assets.NewStore(pack)
	if err != nil {
		return fmt.Errorf("validate embedded assets: %w", err)
	}
	level, err := levels.LoadEmbedded(levels.LevelID(levelID), levels.StoreResolver{Store: store}, nil)
	if err != nil {
		return fmt.Errorf("load embedded level: %w", err)
	}
	fmt.Printf("GoKeen: %d meshes, %d materials, level %s loaded\n", len(pack.Meshes), len(pack.Materials), level.ID)
	if err := sdl2.Init(); err != nil {
		return err
	}
	defer sdl2.Quit()

	window, err := sdl2.CreateWindow(title, width, height, fullscreen)
	if err != nil {
		return err
	}
	defer window.Destroy()
	window.SetRelativeMouse(true)

	renderer, err := vulkan.New(window, width, height)
	if err != nil {
		return err
	}
	defer renderer.Destroy()
	if err := renderer.UploadScene(store, level); err != nil {
		return fmt.Errorf("upload scene: %w", err)
	}

	spawn := level.Spawn.Position.Vector()
	cam := camera{
		position: spawn.Add(maths.V3(0, 3, 12)),
		yaw:      0,
		pitch:    -0.15,
		speed:    12,
	}
	var collision *physics.Mesh
	if len(level.Geometry) > 0 {
		collisionID := level.Geometry[0].Collision
		if collisionID == "" {
			collisionID = level.Geometry[0].Mesh
		}
		if collisionMesh, err := store.Mesh(assets.MeshID(collisionID)); err == nil {
			positions := make([]maths.Vec3, len(collisionMesh.Vertices))
			for i, v := range collisionMesh.Vertices {
				positions[i] = v.Position
			}
			collision = physics.NewMesh(physics.TrianglesFromIndexed(positions, collisionMesh.Indices))
		}
	}
	player := players.New(spawn)
	player.PogoEnabled = pogo
	third := cameras.NewThirdPerson(player.Center())
	var playerInstance *vulkan.MeshInstance
	type activeEnemy struct {
		enemy    *enemies.Enemy
		instance *vulkan.MeshInstance
	}
	var activeEnemies []activeEnemy
	if !fly {
		playerInstance, err = renderer.AddMeshInstance(store, "player_keen")
		if err != nil {
			return fmt.Errorf("add player mesh: %w", err)
		}
		for _, spec := range level.Entities {
			if spec.Type != "yorp" {
				continue
			}
			enemy := enemies.New(spec.Transform.Position.Vector())
			instance, err := renderer.AddMeshInstance(store, "yorp")
			if err != nil {
				return fmt.Errorf("add yorp mesh: %w", err)
			}
			activeEnemies = append(activeEnemies, activeEnemy{enemy: enemy, instance: instance})
		}
	}
	keys := make(map[int]bool)
	mouseButtons := make(map[int]bool)
	var laserInstance *vulkan.MeshInstance
	var laserCooldown, laserVisible float32
	if !fly {
		laserInstance, err = renderer.AddMeshInstance(store, "debug_cube")
		if err != nil {
			return fmt.Errorf("add laser mesh: %w", err)
		}
		laserInstance.SetTransform(maths.M4Scale(maths.V3(0, 0, 0)))
	}
	clock := core.NewWallClock()
	frameCount := 0
	running := true
	for running {
		for {
			event, ok := window.Poll()
			if !ok {
				break
			}
			switch event.Type {
			case sdl2.EventQuit:
				running = false
			case sdl2.EventKey:
				keys[event.Key] = event.Action == 1
				if event.Action == 1 && event.Key == 27 {
					running = false
				}
			case sdl2.EventMouseButton:
				mouseButtons[event.Button] = event.Action == 1
			case sdl2.EventMouseMove:
				if fly {
					cam.yaw -= float32(event.MouseDX) * 0.002
					cam.pitch -= float32(event.MouseDY) * 0.002
					if cam.pitch > 1.4 {
						cam.pitch = 1.4
					}
					if cam.pitch < -1.4 {
						cam.pitch = -1.4
					}
				} else {
					third.Look(float32(event.MouseDX), float32(event.MouseDY))
				}
			case sdl2.EventResize:
				if event.Width > 0 && event.Height > 0 {
					if err := renderer.Resize(event.Width, event.Height); err != nil {
						return err
					}
				}
			}
		}
		if !running {
			break
		}
		dt := float32(clock.Delta())
		if dt > 0.1 {
			dt = 0.1
		}
		w, h := renderer.Extent()
		if fly {
			cam.update(dt, keys)
			if w > 0 && h > 0 {
				cam.setCamera(renderer, float32(w)/float32(h))
			}
		} else {
			input := players.Input{Yaw: third.Yaw}
			if keys[int('w')] || keys[1073741906] {
				input.Forward = 1
			}
			if keys[int('s')] || keys[1073741905] {
				input.Forward = -1
			}
			if keys[int('a')] || keys[1073741904] {
				input.Strafe = -1
			}
			if keys[int('d')] || keys[1073741903] {
				input.Strafe = 1
			}
			input.Jump = keys[int(' ')] || keys[1073742048]
			input.Pogo = input.Jump
			player.Update(dt, input, collision)
			playerInstance.SetTransform(player.ModelMatrix(maths.V3(0, 0.462, 0), 0.8))
			for _, active := range activeEnemies {
				active.enemy.Update(dt, collision)
				active.instance.SetTransform(active.enemy.ModelMatrix(maths.V3(0, 0.331, 0), 0.7))
			}
			if laserCooldown > 0 {
				laserCooldown -= dt
			}
			if laserVisible > 0 {
				laserVisible -= dt
				if laserVisible <= 0 {
					laserInstance.SetTransform(maths.M4Scale(maths.V3(0, 0, 0)))
				}
			}
			if (keys[int('f')] || mouseButtons[1] || autofire) && laserCooldown <= 0 {
				dir := third.Forward()
				origin := player.Position.Add(maths.V3(0, player.Height*0.6, 0)).Add(dir.Scale(0.6))
				hitDist := float32(200)
				if collision != nil {
					if hit, ok := collision.Raycast(maths.NewRay(origin, dir), hitDist); ok {
						hitDist = hit.Distance
					}
				}
				for _, active := range activeEnemies {
					if dist, ok := active.enemy.RayHit(origin, dir, hitDist); ok {
						active.enemy.Stun(2)
						hitDist = dist
					}
				}
				end := origin.Add(dir.Scale(hitDist))
				laserInstance.SetTransform(beamMatrix(origin, end, 0.06))
				laserVisible = 0.12
				laserCooldown = 0.18
				if autofire {
					laserVisible = 0.5
				}
			}
			third.Target = player.Center()
			if w > 0 && h > 0 {
				vp, pos := third.ViewProjection(float32(w) / float32(h))
				renderer.SetCamera(vp, pos)
			}
		}
		if err := renderer.Frame(); err != nil {
			if errors.Is(err, vulkan.ErrOutOfDate) {
				w, h := window.Size()
				if err := renderer.Resize(w, h); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		frameCount++
		if frames > 0 && frameCount >= frames {
			break
		}
		// The FIFO present mode already throttles to the display. A tiny sleep
		// avoids spinning if the driver returns early on a minimized window.
		time.Sleep(time.Millisecond)
	}
	return nil
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
	if keys[303] || keys[304] { // shift variants vary by platform; keep a fallback
		speed *= 3
	}
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
	if c.pitch > 1.4 {
		c.pitch = 1.4
	}
	if c.pitch < -1.4 {
		c.pitch = -1.4
	}
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
		return maths.M4Scale(maths.V3(0, 0, 0))
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
