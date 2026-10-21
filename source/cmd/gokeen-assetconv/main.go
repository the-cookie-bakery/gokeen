package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gokeen/assets"
	"gokeen/levels"
	"gokeen/tools/legacy/gltf"
	"gokeen/tools/legacy/importer"
)

type levelJob struct {
	id        levels.LevelID
	name      string
	source    string
	meshID    string
	collision string
}

func main() {
	game := flag.String("game", "../old_game", "legacy game asset root")
	packPath := flag.String("pack", "assets/pack/gokeen.gkp", "output asset pack")
	levelDir := flag.String("levels", "levels/data", "output directory for level JSON")
	sourceDir := flag.String("sources", "assets/sources/meshes", "output directory for editable glTF meshes")
	reportPath := flag.String("report", "build/asset-report.json", "conversion report path")
	backportPath := flag.String("backport", "levels/backport.json", "backport manifest path")
	force := flag.Bool("force", false, "overwrite existing level JSON")
	flag.Parse()

	if err := run(*game, *packPath, *levelDir, *sourceDir, *reportPath, *backportPath, *force); err != nil {
		fmt.Fprintln(os.Stderr, "asset conversion failed:", err)
		os.Exit(1)
	}
}

func run(game, packPath, levelDir, sourceDir, reportPath, backportPath string, force bool) error {
	conv, err := importer.New(game)
	if err != nil {
		return err
	}
	jobs := []levelJob{
		{id: "map", name: "Map", source: "Map/map2b.b3d", meshID: "map_visual", collision: "map_collision"},
		{id: "level1", name: "Level 1", source: "Levels/Level1/Level1.b3d", meshID: "level1_visual", collision: "level1_collision"},
		{id: "level2", name: "Level 2", source: "Levels/Level2/Level2ba.b3d", meshID: "level2_visual", collision: "level2_collision"},
		{id: "level5", name: "Level 5", source: "Levels/Level5/Level5a_gls.b3d", meshID: "level5_visual", collision: "level5_collision"},
	}
	modelJobs := []struct {
		legacy string
		meshID string
		prefix string
	}{
		{legacy: "Keen3f/Keen3f.3ds", meshID: "player_keen", prefix: "keen"},
		{legacy: "NPC/Yorp1h/YorpStand.3ds", meshID: "yorp", prefix: "yorp"},
		{legacy: "Items/Pogo.3ds", meshID: "item_pogo", prefix: "item_pogo"},
		{legacy: "Items/Gun4.3ds", meshID: "item_gun", prefix: "item_gun"},
		{legacy: "Items/Key.3ds", meshID: "item_key", prefix: "item_key"},
		{legacy: "Items/Pizza.3ds", meshID: "item_pizza", prefix: "item_pizza"},
		{legacy: "Items/Soda/Soda.3ds", meshID: "item_soda", prefix: "item_soda"},
		{legacy: "Items/Sign/ExitSign.3ds", meshID: "exit_sign", prefix: "exit_sign"},
	}
	for _, job := range jobs {
		fmt.Printf("importing %s (%s)\n", job.id, job.source)
		meshID, err := conv.ImportStaticB3D(job.source, job.meshID, string(job.id), true)
		if err != nil {
			return err
		}
		if _, err := conv.ImportCollisionCopy(string(meshID), job.collision); err != nil {
			return err
		}
		if _, err := gltf.ExportMesh(sourceDir, conv.Pack(), meshID); err != nil {
			return fmt.Errorf("export glTF for %s: %w", job.id, err)
		}
		mesh := conv.Pack().Meshes[meshID]
		jsonPath := filepath.Join(levelDir, string(job.id)+".json")
		if err := conv.WriteLevelSeed(job.id, job.name, string(meshID), job.collision, jsonPath, mesh.Bounds.Center(), force); err != nil {
			return fmt.Errorf("write level seed %s: %w", job.id, err)
		}
	}
	conv.AddProceduralAssets()
	for _, job := range modelJobs {
		fmt.Printf("importing model %s (%s)\n", job.meshID, job.legacy)
		meshID, err := conv.ImportStatic3DS(job.legacy, job.meshID, job.prefix, true)
		if err != nil {
			return err
		}
		if _, err := gltf.ExportMesh(sourceDir, conv.Pack(), meshID); err != nil {
			return fmt.Errorf("export glTF for %s: %w", job.meshID, err)
		}
	}
	fmt.Println("importing skinned model player_keen_skinned")
	if _, err := conv.ImportSkinnedB3D("Keen3f/KeenWalk.b3d", "player_keen_skinned", "skel_keen", "keen", []importer.ClipSpec{
		{Name: "walk", File: "Keen3f/KeenWalk.b3d", Loop: true},
		{Name: "shoot", File: "Keen3f/KeenShoot.b3d", Loop: false},
		{Name: "jump", File: "Keen3f/KeenJump.b3d", Loop: false},
	}, true); err != nil {
		return err
	}
	fmt.Println("importing skinned pogo model player_keen_pogo")
	if _, err := conv.ImportKeenPogo("Keen3f/KeenPogo.3ds", "player_keen_pogo", "skel_keen"); err != nil {
		return err
	}
	if err := conv.AddBindPoseClip("clip_keen_pogo", "pogo", 0, true); err != nil {
		return err
	}
	fmt.Println("importing skinned death model player_keen_dead")
	if _, err := conv.ImportSkinnedB3D("Keen3f/KeenDie.b3d", "player_keen_dead", "skel_keen_dead", "keen_dead", []importer.ClipSpec{
		{Name: "die", File: "Keen3f/KeenDie.b3d", Loop: false},
	}, true); err != nil {
		return err
	}
	if err := conv.RetargetClip("clip_keen_dead_die", "skel_keen_dead", "skel_keen", "clip_keen_die", "die", map[string]string{
		"joint1": "PelvisC",
		"joint2": "AnkleR",
		"joint3": "FootR",
		"joint4": "AnkleL",
		"joint5": "FootL",
	}); err != nil {
		return err
	}
	if err := conv.AddDeathSquash("clip_keen_die", "skel_keen", 0.30, 0.15, 1.2); err != nil {
		return err
	}
	if err := conv.Pack().Validate(); err != nil {
		return err
	}
	packData, err := assets.WritePack(conv.Pack())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(packPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(packPath, packData, 0o644); err != nil {
		return err
	}
	report := conv.Report()
	if err := writeReport(reportPath, report); err != nil {
		return err
	}
	if err := writeSourceManifest(filepath.Join(filepath.Dir(sourceDir), "manifest.json"), jobs, modelJobs); err != nil {
		return err
	}
	if err := updateBackport(backportPath, jobs); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes), %d meshes, %d materials, %d textures\n",
		packPath, len(packData), len(conv.Pack().Meshes), len(conv.Pack().Materials), len(conv.Pack().Textures))
	return nil
}

func writeReport(path string, report importer.Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeSourceManifest(path string, jobs []levelJob, models []struct {
	legacy string
	meshID string
	prefix string
}) error {
	manifest := assets.SourceManifest{FormatVersion: assets.SourceManifestVersion}
	for _, job := range jobs {
		manifest.Assets = append(manifest.Assets, assets.SourceAsset{
			ID:           job.meshID,
			Kind:         "mesh",
			Path:         "meshes/" + job.meshID + ".gltf",
			LegacySource: job.source,
			MeshID:       job.meshID,
		})
	}
	for _, job := range models {
		manifest.Assets = append(manifest.Assets, assets.SourceAsset{
			ID:           job.meshID,
			Kind:         "model",
			Path:         "meshes/" + job.meshID + ".gltf",
			LegacySource: job.legacy,
			MeshID:       job.meshID,
		})
	}
	data, err := manifest.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func updateBackport(path string, jobs []levelJob) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	manifest, err := levels.LoadBackport(data)
	if err != nil {
		return err
	}
	imported := make(map[levels.LevelID]bool)
	for _, job := range jobs {
		imported[job.id] = true
	}
	for i := range manifest.Levels {
		if imported[manifest.Levels[i].ID] && manifest.Levels[i].Status == levels.BackportPending {
			manifest.Levels[i].Status = levels.BackportImported
		}
	}
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}
