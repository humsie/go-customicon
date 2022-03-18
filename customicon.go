package go_customicon

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/humsie/go-customicon/apple"
	"github.com/humsie/go-customicon/helpers"
	"github.com/jackmordaunt/icns"
	"github.com/pkg/xattr"
	"image"
	"log"
	"os"
	"strings"
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

func SetCustomIcon(imagePath, targetPath string) {

	var err error
	ci := CustomIcon{}

	err = ci.SetImageFromPath(imagePath)
	if err != nil {
		log.Fatalln("Error:", err.Error())
	}
	err = ci.SetTargetPath(targetPath)
	if err != nil {
		log.Fatalln("Error:", err.Error())
	}

	ci.CreateIconSet()
	ci.WriteExtendedAttributes()
}
func SetCustomIconFromImage(image image.Image, targetPath string) {
	var err error
	ci := CustomIcon{}

	err = ci.SetImage(image)
	if err != nil {
		log.Fatalln("Error:", err.Error())
	}
	err = ci.SetTargetPath(targetPath)
	if err != nil {
		log.Fatalln("Error:", err.Error())
	}

	ci.CreateIconSet()
	ci.WriteExtendedAttributes()
}

func RemoveCustomIcon(targetPath string) {

	var err error
	ci := CustomIcon{}

	err = ci.SetTargetPath(targetPath)
	if err != nil {
		log.Fatalln("Error:", err.Error())
	}

	ci.RemoveExtendedAttributes()
}

type CustomIcon struct {
	targetPath  string
	targetIsDir bool

	imageData image.Image

	iconSet []byte
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
		log.Fatalf("opening source image: %v", err)
	}
	defer pngf.Close()
	ci.imageData, _, err = image.Decode(pngf)
	if err != nil {
		log.Fatalf("decoding source image: %v", err)
	}

	return nil

}

func (ci *CustomIcon) SetTargetPath(path string) error {

	targetInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	ci.targetPath = path
	if targetInfo.IsDir() {
		ci.targetIsDir = true
	}

	return nil
}

func (ci *CustomIcon) CreateIconSet() {

	buffer := bytes.NewBuffer([]byte{})

	if err := icns.Encode(buffer, ci.imageData); err != nil {
		log.Fatalf("encoding iconset failed: %v", err)
	}

	ci.iconSet = buffer.Bytes()

}

func (ci *CustomIcon) ShowAttributes() {

	var src *os.File
	var err error
	if ci.targetIsDir {
		xB := []byte{13}
		src, err = os.Open(ci.targetPath + "/Icon" + string(xB))
	} else {
		src, err = os.Open(ci.targetPath)
	}
	if err != nil {
		log.Fatalf("opening destination file: %v", err)
	}
	defer src.Close()

	attrlist, err := xattr.FList(src)
	if err != nil {
		log.Fatalf("reading xattr: %v", err)
	}
	fmt.Println("Attributes found:\n - ", strings.Join(attrlist, "\n - "))

	attrFinderInfo, err := xattr.FGet(src, "com.apple.FinderInfo")

	fmt.Printf("%v\n", hex.Dump(attrFinderInfo))

	for i := range attrFinderInfo {
		if attrFinderInfo[i] > 0 {
			fmt.Println(i, "=>", attrFinderInfo[i])
		}
	}
}

func (ci *CustomIcon) RemoveExtendedAttributes() {

	if ci.targetIsDir {
		xB := []byte{13}

		err := os.Remove(ci.targetPath + "/Icon" + string(xB))
		if err != nil {
			log.Fatalf("Can't delete special icon file in target folder: %v", err)
		}

		xattr.Remove(ci.targetPath, "com.apple.FinderInfo")
	} else {
		xattr.Remove(ci.targetPath, "com.apple.ResourceFork")
	}

	xattr.Remove(ci.targetPath, "com.apple.FinderInfo")

}
func (ci *CustomIcon) WriteExtendedAttributes() {

	resourceForkData, err := apple.NewResourceForkWithData(ci.iconSet).Bytes()
	if err != nil {
		log.Println("Blaat")
		return
	}

	if ci.targetIsDir {

		xB := []byte{13}
		iconFile, err := os.Create(ci.targetPath + "/Icon" + string(xB))
		if err != nil {
			log.Fatalf("Can't create special icon file in target folder: %v", err)
		}
		defer iconFile.Close()
		xattr.FSet(iconFile, "com.apple.ResourceFork", resourceForkData)

		finderInfoFile := make([]byte, 32)
		temp := helpers.Int32toBytes(Fhidden + FcustomIcon)
		finderInfoFile[8] = temp[2]
		finderInfoFile[9] = temp[3]

		xattr.FSet(iconFile, "com.apple.FinderInfo", finderInfoFile)

		finderInfoFolder := make([]byte, 32)
		temp = helpers.Int32toBytes(FcustomIcon)
		finderInfoFolder[8] = temp[2]
		finderInfoFolder[9] = temp[3]

		xattr.Set(ci.targetPath, "com.apple.FinderInfo", finderInfoFolder)

	} else {

		targetFile, err := os.Open(ci.targetPath)
		if err != nil {
			log.Fatalf("Can't open target file: %v", err)
		}
		defer targetFile.Close()
		xattr.FSet(targetFile, "com.apple.ResourceFork", resourceForkData)

		finderInfoFile := make([]byte, 32)
		temp := helpers.Int32toBytes(FcustomIcon)
		finderInfoFile[8] = temp[2]
		finderInfoFile[9] = temp[3]

		xattr.FSet(targetFile, "com.apple.FinderInfo", finderInfoFile)

	}

}
