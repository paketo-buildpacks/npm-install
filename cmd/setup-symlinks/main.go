package main

import (
	"log"
	"os"

	"github.com/paketo-buildpacks/npm-install/cmd/setup-symlinks/internal"
)

func main() {
	err := internal.Run(os.Args[0], os.Getenv("CNB_APP_DIR"))
	if err != nil {
		log.Fatal(err)
	}
}
