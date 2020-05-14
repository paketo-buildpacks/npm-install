package main

import (
	"github.com/paketo-buildpacks/npm/npm"
	"github.com/paketo-buildpacks/packit"
)

func main() {
	packageJSONParser := npm.NewPackageJSONParser()
	packit.Detect(npm.Detect(packageJSONParser))
}
