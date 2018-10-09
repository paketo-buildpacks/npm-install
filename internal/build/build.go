package build

import (
	"github.com/buildpack/libbuildpack"
)

const NodeDependency = "node"

func CreateLaunchMetadata() libbuildpack.LaunchMetadata {
	return libbuildpack.LaunchMetadata{
		Processes: libbuildpack.Processes{
			libbuildpack.Process{
				Type:    "web",
				Command: "npm start",
			},
		},
	}
}
