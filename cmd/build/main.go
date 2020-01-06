package main

import (
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/pexec"
)

func main() {
	executable := pexec.NewExecutable("npm", lager.NewLogger("npm"))
	nodePackageManager := npm.NewNodePackageManager(executable)

	packit.Build(npm.Build(nodePackageManager))
}
