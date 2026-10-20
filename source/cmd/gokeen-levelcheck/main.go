package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gokeen/assets"
	"gokeen/levels"
)

func main() {
	packPath := flag.String("pack", "assets/pack/gokeen.gkp", "asset pack referenced by levels")
	levelDir := flag.String("levels", "levels/data", "level JSON directory")
	backportPath := flag.String("backport", "levels/backport.json", "backport manifest")
	flag.Parse()

	if err := run(*packPath, *levelDir, *backportPath); err != nil {
		fmt.Fprintln(os.Stderr, "level check failed:", err)
		os.Exit(1)
	}
}

func run(packPath, levelDir, backportPath string) error {
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
		return err
	}
	manifestData, err := os.ReadFile(backportPath)
	if err != nil {
		return err
	}
	manifest, err := levels.LoadBackport(manifestData)
	if err != nil {
		return err
	}
	count := 0
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
		if lvl.ID != entry.ID {
			return fmt.Errorf("manifest id %q does not match JSON id %q", entry.ID, lvl.ID)
		}
		fmt.Printf("%s: %s (%d geometry, %d entities)\n", entry.ID, entry.Status, len(lvl.Geometry), len(lvl.Entities))
		count++
	}
	if count == 0 {
		return fmt.Errorf("no levels checked")
	}
	fmt.Printf("checked %d levels from %s\n", count, levelDir)
	return nil
}
