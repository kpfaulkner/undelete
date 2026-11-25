package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func fakeVolume(t *testing.T, vi volumeInfo, err error) {
	t.Helper()
	old := lookupVolume
	lookupVolume = func(string) (volumeInfo, error) { return vi, err }
	t.Cleanup(func() { lookupVolume = old })
}

func TestCheckWriteTarget(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name    string
		vi      volumeInfo
		lookErr error
		path    string
		wantErr bool
	}{
		{"H and label ok", volumeInfo{Root: `H:\`, Label: "UNDELETE"}, nil, dir, false},
		{"lowercase drive ok", volumeInfo{Root: `h:\`, Label: "UNDELETE"}, nil, dir, false},
		{"not yet existing subdir ok", volumeInfo{Root: `H:\`, Label: "UNDELETE"}, nil, filepath.Join(dir, "a", "b"), false},
		{"wrong drive", volumeInfo{Root: `C:\`, Label: "UNDELETE"}, nil, dir, true},
		{"mount point not drive", volumeInfo{Root: `C:\mnt\vol\`, Label: "UNDELETE"}, nil, dir, true},
		{"wrong label", volumeInfo{Root: `H:\`, Label: "BACKUP"}, nil, dir, true},
		{"label case differs", volumeInfo{Root: `H:\`, Label: "undelete"}, nil, dir, true},
		{"empty label", volumeInfo{Root: `H:\`, Label: ""}, nil, dir, true},
		{"network drive", volumeInfo{Root: `H:\`, Label: "UNDELETE", Remote: true}, nil, dir, true},
		{"lookup fails", volumeInfo{}, errors.New("boom"), dir, true},
		{"empty path", volumeInfo{Root: `H:\`, Label: "UNDELETE"}, nil, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeVolume(t, tc.vi, tc.lookErr)
			err := checkWriteTarget(tc.path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("checkWriteTarget(%q) err = %v, wantErr %v", tc.path, err, tc.wantErr)
			}
		})
	}
}

func TestSafeCreateRefusedCreatesNothing(t *testing.T) {
	dir := t.TempDir()
	fakeVolume(t, volumeInfo{Root: `C:\`, Label: "WINDOWS"}, nil)

	if f, err := safeCreate(filepath.Join(dir, "x.bin"), false); err == nil {
		f.Close()
		t.Fatal("safeCreate succeeded on a disallowed volume")
	}
	sub := filepath.Join(dir, "newdir")
	if err := safeMkdirAll(sub); err == nil {
		t.Fatal("safeMkdirAll succeeded on a disallowed volume")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("refused writes still created %d entries", len(entries))
	}
}

func TestSafeCreateAllowed(t *testing.T) {
	dir := t.TempDir()
	fakeVolume(t, volumeInfo{Root: `H:\`, Label: "UNDELETE"}, nil)

	path := filepath.Join(dir, "x.bin")
	f, err := safeCreate(path, false)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := safeCreate(path, false); err == nil {
		t.Fatal("expected failure overwriting without overwrite=true")
	}
	f, err = safeCreate(path, true)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}
