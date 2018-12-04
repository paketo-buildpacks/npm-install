package node

import (
	"encoding/json"
	"os"
)

const Dependency = "node"

type packageJSON struct {
	Engines engines `json:"engines"`
}

type engines struct {
	Node string `json:"node"`
}

func GetNodeVersion(packageFile string) (version string, err error) {
	file, err := os.Open(packageFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	pkgJSON := packageJSON{}
	if err := json.NewDecoder(file).Decode(&pkgJSON); err != nil {
		return "", err
	}

	return pkgJSON.Engines.Node, nil
}
