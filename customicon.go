package go_customicon

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/humsie/go-customicon/apple"
	"github.com/jackmordaunt/icns"
	"github.com/pkg/xattr"
)

var (
	PATH = ""
)

const (
	FonDesktop         = 1 << iota //D | d    Located on the desktop (allowed on folders)
	Fbit2                          // Unknown
	FextHidden                     //E | e    Extension is hidden (allowed on folders)
	Fbit4                          // Unknown
	Fbit5                          // Unknown
	Fbit6                          // Unknown
	Fshared                        // M | m    Shared (can run multiple times)
	FnoInitResource                // N | n    File has no INIT resource
	Finited                        // I | i    Inited - Finder is aware of this file and has given it a location in a window. (allowed on folders)
	Fbit10                         // Unknown
	FcustomIcon                    // C | c    Custom icon (allowed on folders)
	FstationaryPadFile             // T | t    "Stationery Pad" file
	FsystemFile                    // S | s    System file (name locked)
	FhasBundle                     // B | b    Has bundle
	Fhidden                        // V | v    Invisible (allowed on folders)
	FaliasFile                     // A | a    Alias file
)

const (
	attrFinderInfo   = "com.apple.FinderInfo"
	attrResourceFork = "com.apple.ResourceFork"

	finderInfoLength = 32
	// finderFlagsOffset is where the 16-bit big-endian Finder flags live
	// inside the 32-byte FinderInfo.
	finderFlagsOffset = 8

	// iconFileName is the hidden file that carries a folder's custom icon.
	iconFileName = "Icon\r"
)

var (
	defaultLayout = LayoutXattr
)

// Layout is how the icon ends up on disk.
type Layout int

const (
	// LayoutXattr writes com.apple.* extended attributes, as macOS does on
	// its own disks and on an SMB mount.
	LayoutXattr Layout = iota
	// LayoutSynology writes the files Synology DSM keeps those attributes
	// in (@eaDir/<name>@SynoResource), for a process that writes straight
	// to the volume instead of through SMB.
	LayoutSynology
)

func (l Layout) String() string {
	switch l {
	case LayoutXattr:
		return "xattr"
	case LayoutSynology:
		return "synology"
	}
	return fmt.Sprintf("Layout(%d)", int(l))
}

func ParseLayout(s string) (Layout, error) {
	switch s {
	case "xattr":
		return LayoutXattr, nil
	case "synology":
		return LayoutSynology, nil
	}
	return 0, fmt.Errorf("unknown layout %q (want xattr or synology)", s)
}

func SetLayout(layout Layout) {
	defaultLayout = layout
}

func NewCustomIcon() CustomIcon {
	return CustomIcon{layout: defaultLayout}
}

func SetCustomIcon(imagePath, targetPath string) error {
	ci := NewCustomIcon()

	if err := ci.SetImageFromPath(imagePath); err != nil {
		return err
	}
	if err := ci.SetTargetPath(targetPath); err != nil {
		return err
	}
	if err := ci.CreateIconSet(); err != nil {
		return err
	}
	return ci.WriteExtendedAttributes()
}

func SetCustomIconFromIconset(icnsPath, targetPath string) error {
	ci := NewCustomIcon()

	if err := ci.SetIconSetFromPath(icnsPath); err != nil {
		return err
	}
	if err := ci.SetTargetPath(targetPath); err != nil {
		return err
	}
	return ci.WriteExtendedAttributes()
}

func SetCustomIconFromImage(image image.Image, targetPath string) error {
	ci := NewCustomIcon()

	if err := ci.SetImage(image); err != nil {
		return err
	}
	if err := ci.SetTargetPath(targetPath); err != nil {
		return err
	}
	if err := ci.CreateIconSet(); err != nil {
		return err
	}
	return ci.WriteExtendedAttributes()
}

func RemoveCustomIcon(targetPath string) error {
	ci := NewCustomIcon()

	if err := ci.SetTargetPath(targetPath); err != nil {
		return err
	}
	return ci.RemoveExtendedAttributes()
}

type CustomIcon struct {
	targetPath  string
	targetIsDir bool
	layout      Layout

	imageData image.Image

	iconSet []byte
}

func (ci *CustomIcon) SetLayout(layout Layout) {
	ci.layout = layout
}

func (ci *CustomIcon) SetImage(image image.Image) error {
	ci.imageData = image
	return nil
}

func (ci *CustomIcon) SetImageFromPath(path string) error {

	imageInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if imageInfo.IsDir() {
		return errors.New("ImagePath cant be a directory")
	}

	pngf, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening source image: %w", err)
	}
	defer pngf.Close()
	ci.imageData, _, err = image.Decode(pngf)
	if err != nil {
		return fmt.Errorf("decoding source image: %w", err)
	}

	return nil

}

func (ci *CustomIcon) SetIconSetFromPath(path string) error {

	imageInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if imageInfo.IsDir() {
		return errors.New("ImagePath cant be a directory")
	}

	icnsdata, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("opening source iconset: %w", err)
	}

	ci.iconSet = icnsdata

	return nil

}

func (ci *CustomIcon) SetTargetPath(path string) error {

	targetInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	ci.targetPath = path
	ci.targetIsDir = targetInfo.IsDir()

	return nil
}

func (ci *CustomIcon) CreateIconSet() error {

	if ci.imageData == nil {
		return errors.New("no image set")
	}

	buffer := bytes.NewBuffer([]byte{})

	if err := icns.Encode(buffer, ci.imageData); err != nil {
		return fmt.Errorf("encoding iconset: %w", err)
	}

	ci.SetIconSet(buffer.Bytes())

	return nil
}
func (ci *CustomIcon) SetIconSet(data []byte) {
	ci.iconSet = data
}
func (ci *CustomIcon) GetIconSet() []byte {
	return ci.iconSet
}

func (ci *CustomIcon) iconFilePath() string {
	if ci.layout == LayoutSynology {
		return ci.targetPath + "/" + synoIconFileName
	}
	return ci.targetPath + "/" + iconFileName
}

func (ci *CustomIcon) ShowAttributes() error {

	if ci.layout == LayoutSynology {
		return ci.showSynology()
	}

	path := ci.targetPath
	if ci.targetIsDir {
		path = ci.iconFilePath()
	}

	attrlist, err := xattr.List(path)
	if err != nil {
		return fmt.Errorf("reading xattr list of %s: %w", path, err)
	}
	fmt.Println("Attributes found:\n - ", strings.Join(attrlist, "\n - "))

	finderInfo, err := xattr.Get(path, attrFinderInfo)
	if err != nil {
		return fmt.Errorf("reading %s of %s: %w", attrFinderInfo, path, err)
	}

	fmt.Printf("%v\n", hex.Dump(finderInfo))

	for i := range finderInfo {
		if finderInfo[i] > 0 {
			fmt.Println(i, "=>", finderInfo[i])
		}
	}

	return nil
}

func (ci *CustomIcon) RemoveExtendedAttributes() error {

	if ci.layout == LayoutSynology {
		return ci.removeSynology()
	}

	if ci.targetIsDir {
		err := os.Remove(ci.iconFilePath())
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("deleting icon file in target folder: %w", err)
		}
	} else {
		err := xattr.Remove(ci.targetPath, attrResourceFork)
		if err != nil && !isNoAttr(err) {
			return fmt.Errorf("removing %s: %w", attrResourceFork, err)
		}
	}

	return clearFinderFlags(ci.targetPath, FcustomIcon)
}

func (ci *CustomIcon) WriteExtendedAttributes() error {

	if len(ci.iconSet) == 0 {
		return errors.New("no iconset set")
	}

	resourceForkData, err := apple.NewResourceForkWithData(ci.iconSet).Bytes()
	if err != nil {
		return fmt.Errorf("building resource fork: %w", err)
	}

	if ci.layout == LayoutSynology {
		return ci.writeSynology(resourceForkData)
	}

	if !ci.targetIsDir {
		if err := xattr.Set(ci.targetPath, attrResourceFork, resourceForkData); err != nil {
			return fmt.Errorf("writing %s: %w", attrResourceFork, err)
		}
		return setFinderFlags(ci.targetPath, FcustomIcon)
	}

	iconPath := ci.iconFilePath()
	iconFile, err := os.Create(iconPath)
	if err != nil {
		return fmt.Errorf("creating icon file in target folder: %w", err)
	}
	if err := iconFile.Close(); err != nil {
		return fmt.Errorf("closing icon file in target folder: %w", err)
	}

	if err := xattr.Set(iconPath, attrResourceFork, resourceForkData); err != nil {
		return fmt.Errorf("writing %s on icon file: %w", attrResourceFork, err)
	}
	if err := setFinderFlags(iconPath, Fhidden|FcustomIcon); err != nil {
		return fmt.Errorf("icon file: %w", err)
	}
	if err := setFinderFlags(ci.targetPath, FcustomIcon); err != nil {
		return fmt.Errorf("target folder: %w", err)
	}

	return nil
}

// setFinderFlags ORs flags into the Finder flags of path, keeping the rest of
// an existing FinderInfo (labels, other flags) intact.
func setFinderFlags(path string, flags uint16) error {
	finderInfo, err := readFinderInfo(path)
	if err != nil {
		return err
	}

	finderInfo = withFinderFlags(finderInfo, flags, 0)

	if err := xattr.Set(path, attrFinderInfo, finderInfo); err != nil {
		return fmt.Errorf("writing %s: %w", attrFinderInfo, err)
	}
	return nil
}

// clearFinderFlags clears flags in the Finder flags of path. A FinderInfo that
// ends up all zeroes is removed rather than written back.
func clearFinderFlags(path string, flags uint16) error {
	finderInfo, err := readFinderInfo(path)
	if err != nil {
		return err
	}

	finderInfo = withFinderFlags(finderInfo, 0, flags)

	if isZeroFinderInfo(finderInfo) {
		err = xattr.Remove(path, attrFinderInfo)
		if err != nil && !isNoAttr(err) {
			return fmt.Errorf("removing %s: %w", attrFinderInfo, err)
		}
		return nil
	}

	if err := xattr.Set(path, attrFinderInfo, finderInfo); err != nil {
		return fmt.Errorf("writing %s: %w", attrFinderInfo, err)
	}
	return nil
}

// readFinderInfo returns the FinderInfo of path, padded to 32 bytes, or a
// zeroed one when the attribute does not exist yet.
func readFinderInfo(path string) ([]byte, error) {
	finderInfo := make([]byte, finderInfoLength)

	existing, err := xattr.Get(path, attrFinderInfo)
	if err != nil {
		if isNoAttr(err) {
			return finderInfo, nil
		}
		return nil, fmt.Errorf("reading %s: %w", attrFinderInfo, err)
	}

	copy(finderInfo, existing)
	return finderInfo, nil
}

// withFinderFlags sets and clears bits in the Finder flags of finderInfo,
// padding it to 32 bytes first. Every other byte is left as it was.
func withFinderFlags(finderInfo []byte, set, clear uint16) []byte {
	if len(finderInfo) < finderInfoLength {
		padded := make([]byte, finderInfoLength)
		copy(padded, finderInfo)
		finderInfo = padded
	}

	current := binary.BigEndian.Uint16(finderInfo[finderFlagsOffset:])
	binary.BigEndian.PutUint16(finderInfo[finderFlagsOffset:], (current|set)&^clear)

	return finderInfo
}

func isZeroFinderInfo(finderInfo []byte) bool {
	return bytes.Count(finderInfo, []byte{0}) == len(finderInfo)
}

func isNoAttr(err error) bool {
	return errors.Is(err, xattr.ENOATTR)
}
