package main

import (
	"flag"
	"fmt"
	customicon "github.com/humsie/go-customicon"
	"log"
	"os"
	"path/filepath"
)

const (
	PATH = "downloads/5184444/images/"
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

	fmt.Println("TargetPath: %s", targetPath)
	fmt.Println("ImagePath: %s", imagePath)

	customicon.SetCustomIcon(imagePath, targetPath)

}
