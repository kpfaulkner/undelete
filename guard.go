package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Write safeguard.
//
// This tool must never write to a drive the user cares about. A write is only
// allowed when BOTH of these hold for the destination:
//
//  1. it lives on drive H:
//  2. the volume label of that drive is exactly "UNDELETE"
//
// Every write in the program MUST go through safeCreate / safeMkdirAll below
// (or call checkWriteTarget first). Do not call os.Create, os.OpenFile with a
// write flag, os.MkdirAll, os.WriteFile etc. anywhere else. The source volume
// is only ever opened read-only.
const (
	allowedDrive = "H:"
	allowedLabel = "UNDELETE"
)

// volumeInfo describes the volume that holds a path.
type volumeInfo struct {
	Root   string // volume root, e.g. `H:\`, or a mount point like `C:\mnt\vol\`
	Label  string // volume label
	Remote bool   // network drive
}

// lookupVolume returns info for the volume holding an existing path. It is a
// variable so tests can replace it; the real implementation is per-OS.
var lookupVolume = platformLookupVolume

// checkWriteTarget returns nil only if writing at path is allowed. path may
// not exist yet (output dirs are created later); the nearest existing ancestor
// is used, with symlinks and junctions resolved so a link on H: that points
// at another drive is rejected.
func checkWriteTarget(path string) error {
	if path == "" {
		return fmt.Errorf("write refused: empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("write refused: cannot resolve %q: %w", path, err)
	}
	real, err := resolveExisting(abs)
	if err != nil {
		return fmt.Errorf("write refused: cannot resolve %q: %w", path, err)
	}
	vi, err := lookupVolume(real)
	if err != nil {
		return fmt.Errorf("write refused: cannot identify volume of %q: %w", path, err)
	}
	if vi.Remote {
		return fmt.Errorf("write refused: %q is on a network drive", path)
	}
	root := strings.TrimRight(vi.Root, `\/`)
	if !strings.EqualFold(root, allowedDrive) {
		return fmt.Errorf("write refused: %q is on volume %q, writes are only allowed on %s",
			path, vi.Root, allowedDrive)
	}
	if vi.Label != allowedLabel {
		return fmt.Errorf("write refused: volume %s is labelled %q, it must be labelled %q",
			allowedDrive, vi.Label, allowedLabel)
	}
	return nil
}

// resolveExisting walks up from p to the nearest existing path and returns it
// with symlinks resolved.
func resolveExisting(p string) (string, error) {
	for {
		if _, err := os.Lstat(p); err == nil {
			return filepath.EvalSymlinks(p)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return "", fmt.Errorf("no existing ancestor")
		}
		p = parent
	}
}

// safeMkdirAll creates dir (and parents) only if writes are allowed there.
func safeMkdirAll(dir string) error {
	if err := checkWriteTarget(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// safeCreate creates path for writing only if writes are allowed there. With
// overwrite false it fails if the file already exists.
func safeCreate(path string, overwrite bool) (*os.File, error) {
	if err := checkWriteTarget(filepath.Dir(path)); err != nil {
		return nil, err
	}
	flags := os.O_WRONLY | os.O_CREATE
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	return os.OpenFile(path, flags, 0o644)
}
