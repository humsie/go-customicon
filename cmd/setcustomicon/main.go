package main

import (
	"flag"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	customicon "github.com/humsie/go-customicon"
)

func main() {
	flagTarget := flag.String("target", "", "")
	flagImage := flag.String("image", "", "")
	flagClear := flag.Bool("clear", false, "remove the custom icon from target")
	flagLayout := flag.String("layout", "xattr", "how the icon is stored: xattr (macOS, SMB mount) or synology (directly on a DSM volume)")

	flag.Parse()

	var targetPath, imagePath string
	var err error

	layout, err := customicon.ParseLayout(*flagLayout)
	if err != nil {
		log.Fatalf("Error: %s", err.Error())
	}
	customicon.SetLayout(layout)

	if *flagTarget == "" {
		targetPath, err = os.Getwd()
		if err != nil {
			log.Fatalf("Error getting current working dir: %s", err.Error())
		}
	} else {
		targetPath, err = filepath.Abs(*flagTarget)
		if err != nil {
			log.Fatalf("Error getting current working dir: %s", err.Error())
		}
	}

	if *flagClear {
		if *flagImage != "" {
			log.Fatalf("-clear and -image can't be combined")
		}

		fmt.Printf("TargetPath: %s\n", targetPath)

		err = customicon.RemoveCustomIcon(targetPath)
		if err != nil {
			log.Fatalf("Error removing custom icon: %s", err.Error())
		}
		return
	}

	if *flagImage == "" {
		log.Fatalf("No image selected")
	}

	imagePath, _ = filepath.Abs(*flagImage)

	fmt.Printf("TargetPath: %s\n", targetPath)
	fmt.Printf("ImagePath: %s\n", imagePath)

	if strings.HasSuffix(imagePath, ".icns") {
		err = customicon.SetCustomIconFromIconset(imagePath, targetPath)
	} else {
		err = customicon.SetCustomIcon(imagePath, targetPath)
	}
	if err != nil {
		log.Fatalf("Error setting custom icon: %s", err.Error())
	}

}
