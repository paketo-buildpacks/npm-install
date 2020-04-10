package main

import (
	"github.com/cloudfoundry/packit"
	"github.com/paketo-buildpacks/npm/npm"
)

func main() {
	packageJSONParser := npm.NewPackageJSONParser()
	packit.Detect(npm.Detect(packageJSONParser))
}
