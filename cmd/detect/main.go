package main

import (
	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/packit"
)

func main() {
	packageJSONParser := npm.NewPackageJSONParser()
	packit.Detect(npm.Detect(packageJSONParser))
}
