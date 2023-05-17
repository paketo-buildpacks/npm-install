package main

import (
	"log"
	"os"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/npm-install/cmd/setup-symlinks/internal"
)

func main() {
	projectPath, set := os.LookupEnv("NODE_PROJECT_PATH")
	if !set {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
	}

	err := internal.Run(os.Args[0], projectPath, npminstall.NewLinkedModuleResolver(npminstall.NewLinker(os.TempDir())))
	if err != nil {
		log.Fatal(err)
	}
}
