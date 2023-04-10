package npminstall

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

type Linker struct {
	tmpDir string
	path   string
}

func NewLinker(tmpDir string) Linker {
	return Linker{
		tmpDir: tmpDir,
	}
}

func (l Linker) WithPath(path string) Symlinker {
	l.path = path
	return l
}

func (l Linker) Link(source, target string) error {
	indirection := filepath.Join(l.tmpDir, fmt.Sprintf("%x", sha256.Sum256([]byte(source+target))))
	if l.path != "" {
		indirection = filepath.Join(l.tmpDir, l.path)
	}

	err := os.RemoveAll(source)
	if err != nil {
		return fmt.Errorf("failed to remove link source: %w", err)
	}

	err = os.RemoveAll(indirection)
	if err != nil {
		return fmt.Errorf("failed to remove link indirection: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(indirection), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create indirection path: %w", err)
	}

	err = os.Symlink(indirection, source)
	if err != nil {
		return fmt.Errorf("failed to create link source: %w", err)
	}

	err = os.Symlink(target, indirection)
	if err != nil {
		return fmt.Errorf("failed to create link indirection: %w", err)
	}

	return nil
}
