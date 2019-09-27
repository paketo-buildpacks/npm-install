package detection

import (
	"encoding/json"
	"fmt"
	"os"
)

func GetNodeVersion(path string) (string, error) {
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
	if err := json.NewDecoder(file).Decode(&pkg); err != nil {
		return "", fmt.Errorf("unable to parse package.json: %s", err)
	}

	return pkg.Engines.Node, nil
}
