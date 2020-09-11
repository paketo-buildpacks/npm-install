package npminstall

import (
	"encoding/json"
	"os"
)

type PackageJSONParser struct{}

func NewPackageJSONParser() PackageJSONParser {
	return PackageJSONParser{}
}

func (p PackageJSONParser) ParseVersion(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}

	err = json.NewDecoder(file).Decode(&pkg)
	if err != nil {
		return "", err
	}

	return pkg.Engines.Node, nil
}
