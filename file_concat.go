package npminstall

import (
	"fmt"
	"io/ioutil"
)

type FileConcat struct{}

func NewFileConcat() FileConcat {
	return FileConcat{}
}

func (c FileConcat) Concat(files ...string) (string, error) {
	tempFile, err := ioutil.TempFile("", "fileconcat*")
	if err != nil {
		return "", fmt.Errorf("could not create temp file: %w", err)
	}

	for _, filename := range files {
		contents, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", fmt.Errorf("could not read file: %w", err)
		}

		if _, err := tempFile.Write(contents); err != nil {
			return "", fmt.Errorf("could not write to temp file: %w", err)
		}
	}

	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("could not close temp file: %w", err)
	}

	return tempFile.Name(), nil
}
