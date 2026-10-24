package go_customicon

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/humsie/go-customicon/apple"
)

// Synology DSM does not keep Apple metadata in extended attributes. Its SMB
// server stores it next to the file, in an @eaDir folder in the same
// directory:
//
//	<dir>/@eaDir/<name>@SynoResource   AppleDouble: FinderInfo + resource fork
//	<dir>/@eaDir/<name>@SynoEAStream   AppleDouble: the other com.apple.* xattrs
//
// A folder's own metadata therefore lives in the @eaDir of its parent. Only
// @SynoResource matters for a custom icon. The SMB server also maps the \r in
// "Icon\r" to U+F00D on disk, so that is the name the file needs here.
const (
	synoEADir          = "@eaDir"
	synoResourceSuffix = "@SynoResource"
	synoEAStreamSuffix = "@SynoEAStream"

	synoIconFileName = "Icon"
)

func synoSidecarPath(path, suffix string) string {
	path = filepath.Clean(path)
	return filepath.Join(filepath.Dir(path), synoEADir, filepath.Base(path)+suffix)
}

// updateSynoResource reads the @SynoResource of path (or starts an empty one),
// applies update and writes it back. A result that holds nothing but zeroes is
// removed instead, so clearing the last flag leaves no stray file behind.
func updateSynoResource(path string, update func(ad *apple.AppleDouble)) error {
	sidecar := synoSidecarPath(path, synoResourceSuffix)

	ad := apple.NewAppleDouble()
	data, err := os.ReadFile(sidecar)
	switch {
	case err == nil:
		ad, err = apple.ParseAppleDouble(data)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", sidecar, err)
		}
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("reading %s: %w", sidecar, err)
	}

	update(ad)

	if isEmptyAppleDouble(ad) {
		err := os.Remove(sidecar)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("removing %s: %w", sidecar, err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(sidecar), 0o777); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(sidecar), err)
	}
	if err := os.WriteFile(sidecar, ad.Bytes(), 0o666); err != nil {
		return fmt.Errorf("writing %s: %w", sidecar, err)
	}
	return nil
}

func isEmptyAppleDouble(ad *apple.AppleDouble) bool {
	for _, e := range ad.Entries {
		if bytes.Count(e.Data, []byte{0}) != len(e.Data) {
			return false
		}
	}
	return true
}

// synoSetIcon puts flags and resourceFork into the @SynoResource of path. The
// FinderInfo entry is set first, so a new file gets the same layout DSM writes.
func synoSetIcon(path string, flags uint16, resourceFork []byte) error {
	return updateSynoResource(path, func(ad *apple.AppleDouble) {
		ad.SetEntry(apple.EntryFinderInfo, withFinderFlags(ad.Entry(apple.EntryFinderInfo), flags, 0))
		ad.SetEntry(apple.EntryResourceFork, resourceFork)
	})
}

func (ci *CustomIcon) writeSynology(resourceForkData []byte) error {

	if !ci.targetIsDir {
		return synoSetIcon(ci.targetPath, FcustomIcon, resourceForkData)
	}

	iconPath := ci.iconFilePath()
	iconFile, err := os.Create(iconPath)
	if err != nil {
		return fmt.Errorf("creating icon file in target folder: %w", err)
	}
	if err := iconFile.Close(); err != nil {
		return fmt.Errorf("closing icon file in target folder: %w", err)
	}

	if err := synoSetIcon(iconPath, Fhidden|FcustomIcon, resourceForkData); err != nil {
		return fmt.Errorf("icon file: %w", err)
	}

	// The folder only carries the flag; its resource fork stays as it is
	// (DSM writes an empty one).
	err = updateSynoResource(ci.targetPath, func(ad *apple.AppleDouble) {
		ad.SetEntry(apple.EntryFinderInfo, withFinderFlags(ad.Entry(apple.EntryFinderInfo), FcustomIcon, 0))
		if !ad.HasEntry(apple.EntryResourceFork) {
			ad.SetEntry(apple.EntryResourceFork, nil)
		}
	})
	if err != nil {
		return fmt.Errorf("target folder: %w", err)
	}

	return nil
}

func (ci *CustomIcon) removeSynology() error {

	clearIcon := func(ad *apple.AppleDouble) {
		if ad.HasEntry(apple.EntryFinderInfo) {
			ad.SetEntry(apple.EntryFinderInfo, withFinderFlags(ad.Entry(apple.EntryFinderInfo), 0, FcustomIcon))
		}
	}

	if !ci.targetIsDir {
		return updateSynoResource(ci.targetPath, func(ad *apple.AppleDouble) {
			clearIcon(ad)
			if ad.HasEntry(apple.EntryResourceFork) {
				ad.SetEntry(apple.EntryResourceFork, nil)
			}
		})
	}

	iconPath := ci.iconFilePath()
	for _, path := range []string{
		iconPath,
		synoSidecarPath(iconPath, synoResourceSuffix),
		synoSidecarPath(iconPath, synoEAStreamSuffix),
	} {
		err := os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("deleting %s: %w", path, err)
		}
	}

	if err := updateSynoResource(ci.targetPath, clearIcon); err != nil {
		return fmt.Errorf("target folder: %w", err)
	}

	return nil
}

func (ci *CustomIcon) showSynology() error {

	path := ci.targetPath
	if ci.targetIsDir {
		path = ci.iconFilePath()
	}

	sidecar := synoSidecarPath(path, synoResourceSuffix)
	data, err := os.ReadFile(sidecar)
	if err != nil {
		return fmt.Errorf("reading %s: %w", sidecar, err)
	}
	ad, err := apple.ParseAppleDouble(data)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", sidecar, err)
	}

	fmt.Printf("%s (filler %q):\n", sidecar, ad.Filler[:])
	for _, e := range ad.Entries {
		fmt.Printf(" - entry %d: %d bytes\n", e.ID, len(e.Data))
	}

	if finderInfo := ad.Entry(apple.EntryFinderInfo); finderInfo != nil {
		fmt.Printf("%v\n", hex.Dump(finderInfo))
	}

	return nil
}
