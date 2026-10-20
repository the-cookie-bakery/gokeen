package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gokeen/assets"
	"gokeen/levels"
	"gokeen/tools/legacy/verify"
)

func main() {
	packPath := flag.String("pack", "assets/pack/gokeen.gkp", "asset pack to verify")
	levelDir := flag.String("levels", "levels/data", "level JSON directory")
	backportPath := flag.String("backport", "levels/backport.json", "backport manifest")
	sourceDir := flag.String("sources", "assets/sources/meshes", "glTF source directory")
	manifestPath := flag.String("manifest", "assets/sources/manifest.json", "source asset manifest")
	flag.Parse()

	if err := run(*packPath, *levelDir, *backportPath, *sourceDir, *manifestPath); err != nil {
		fmt.Fprintln(os.Stderr, "asset check failed:", err)
		os.Exit(1)
	}
}

func run(packPath, levelDir, backportPath, sourceDir, manifestPath string) error {
	packData, err := os.ReadFile(packPath)
	if err != nil {
		return err
	}
	pack, err := assets.ReadPack(packData)
	if err != nil {
		return fmt.Errorf("read pack: %w", err)
	}
	store, err := assets.NewStore(pack)
	if err != nil {
		return fmt.Errorf("validate pack: %w", err)
	}
	fmt.Printf("pack: %d meshes, %d materials, %d textures, %d skeletons, %d clips\n",
		len(pack.Meshes), len(pack.Materials), len(pack.Textures), len(pack.Skeletons), len(pack.Clips))

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	sourceManifest, err := assets.LoadSourceManifest(manifestBytes)
	if err != nil {
		return err
	}
	sourceRoot := filepath.Dir(manifestPath)
	for _, asset := range sourceManifest.Assets {
		if asset.Kind == "mesh" {
			if _, ok := pack.Meshes[assets.MeshID(asset.MeshID)]; !ok {
				return fmt.Errorf("source manifest references missing mesh %q", asset.MeshID)
			}
		}
		if _, err := os.Stat(filepath.Join(sourceRoot, filepath.FromSlash(asset.Path))); err != nil {
			return fmt.Errorf("source asset %q: %w", asset.ID, err)
		}
	}
	fmt.Printf("source manifest: %d assets\n", len(sourceManifest.Assets))

	gltfs, err := filepath.Glob(filepath.Join(sourceDir, "*.gltf"))
	if err != nil {
		return err
	}
	if len(gltfs) == 0 {
		return fmt.Errorf("no glTF source files found in %s", sourceDir)
	}
	totalVerts, totalTris := 0, 0
	for _, path := range gltfs {
		rep, err := verify.VerifyGLTF(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		fmt.Printf("gltf: %s vertices=%d triangles=%d primitives=%d\n", filepath.Base(path), rep.Vertices, rep.Triangles, rep.Primitives)
		totalVerts += rep.Vertices
		totalTris += rep.Triangles
	}
	if totalVerts == 0 || totalTris == 0 {
		return fmt.Errorf("glTF sources contain no geometry")
	}

	manifestData, err := os.ReadFile(backportPath)
	if err != nil {
		return err
	}
	manifest, err := levels.LoadBackport(manifestData)
	if err != nil {
		return err
	}
	verifiedLevels := 0
	for _, entry := range manifest.Levels {
		if entry.Status == levels.BackportMissingSource {
			continue
		}
		path := filepath.Join(filepath.Dir(backportPath), filepath.FromSlash(entry.JSONPath))
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("level %s: %w", entry.ID, err)
		}
		lvl, err := levels.Load(data, levels.StoreResolver{Store: store}, nil)
		if err != nil {
			return fmt.Errorf("level %s: %w", entry.ID, err)
		}
		fmt.Printf("level: %s (%s) geometry=%d entities=%d\n", lvl.ID, entry.Status, len(lvl.Geometry), len(lvl.Entities))
		verifiedLevels++
	}
	if verifiedLevels == 0 {
		return fmt.Errorf("no levels verified")
	}
	fmt.Printf("verified %d levels, %d glTF files, %d vertices, %d triangles\n", verifiedLevels, len(gltfs), totalVerts, totalTris)
	return nil
}
