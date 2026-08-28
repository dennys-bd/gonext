package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveDest returns the destination directory for a scaffolded
// project: path if given, otherwise ./name.
func ResolveDest(name, path string) (string, error) {
	if path != "" {
		return path, nil
	}
	return filepath.Join(".", name), nil
}

// CheckEmpty returns an error if dest exists and is non-empty. A
// missing directory is not an error since Copy creates it.
func CheckEmpty(dest string) error {
	entries, err := os.ReadDir(dest)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("destination %q already exists and is not empty", dest)
	}
	return nil
}
