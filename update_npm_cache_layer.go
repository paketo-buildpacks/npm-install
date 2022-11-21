package npminstall

import (
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func UpdateNpmCacheLayer(logger scribe.Emitter, workingDir string, cacheLayer packit.Layer) (packit.Layer, error) {
	npmCachePath := filepath.Join(workingDir, "npm-cache")
	sum, err := fs.NewChecksumCalculator().Sum(npmCachePath)
	if err != nil {
		return packit.Layer{}, err
	}

	cacheSha, ok := cacheLayer.Metadata["cache_sha"].(string)
	if !ok || sum != cacheSha {
		if err != nil {
			return packit.Layer{}, err
		}

		err = fs.Move(npmCachePath, cacheLayer.Path)
		if err != nil {
			return packit.Layer{}, err
		}

		cacheLayer.Metadata = map[string]interface{}{
			"cache_sha": sum,
		}
	} else {
		logger.Process("Reusing cached layer %s", cacheLayer.Path)
		err = os.RemoveAll(npmCachePath)
		if err != nil {
			return packit.Layer{}, err
		}
	}

	return cacheLayer, nil
}
