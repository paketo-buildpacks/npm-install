package utils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libjavabuildpack"
)

func CopyDirectory(srcDir, destDir string) error {
	destExists, _ := libjavabuildpack.FileExists(destDir)
	if !destExists {
		return errors.New("destination dir must exist")
	}

	files, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		src := filepath.Join(srcDir, f.Name())
		dest := filepath.Join(destDir, f.Name())

		if m := f.Mode(); m&os.ModeSymlink != 0 {
			target, err := os.Readlink(src)
			if err != nil {
				return fmt.Errorf("Error while reading symlink '%s': %v", src, err)
			}
			if err := os.Symlink(target, dest); err != nil {
				return fmt.Errorf("Error while creating '%s' as symlink to '%s': %v", dest, target, err)
			}
		} else if f.IsDir() {
			err = os.MkdirAll(dest, f.Mode())
			if err != nil {
				return err
			}
			if err := CopyDirectory(src, dest); err != nil {
				return err
			}
		} else {
			rc, err := os.Open(src)
			if err != nil {
				return err
			}

			err = libjavabuildpack.WriteToFile(rc, dest, f.Mode())
			if err != nil {
				rc.Close()
				return err
			}
			rc.Close()
		}
	}

	return nil
}

func CopyFile(srcPath, dstPath string) error {
	src, err := ioutil.ReadFile(srcPath)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dstPath, src, 066)
	if err != nil {
		return err
	}

	return nil
}
