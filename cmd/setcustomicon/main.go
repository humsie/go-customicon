package main

import (
	"flag"
	"fmt"
	customicon "github.com/humsie/go-customicon"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	flagTarget := flag.String("target", "", "")
	flagImage := flag.String("image", "", "")

	flag.Parse()

	var targetPath, imagePath string
	var err error

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

	if *flagImage == "" {
		log.Fatalf("No image selected")
	}

	imagePath, _ = filepath.Abs(*flagImage)

	fmt.Printf("TargetPath: %s\n", targetPath)
	fmt.Printf("ImagePath: %s\n", imagePath)

	if strings.HasSuffix(imagePath, ".icns") {
		customicon.SetCustomIconFromIconset(imagePath, targetPath)
	} else {
		customicon.SetCustomIcon(imagePath, targetPath)
	}

}
