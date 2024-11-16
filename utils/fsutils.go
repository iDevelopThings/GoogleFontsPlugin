package utils

import (
	"os"
	"path"
)

func EnsurePathExists(p string) error {
	// if path contains a file, extension exists(?)
	// remove the file name from the path
	if path.Ext(p) != "" {
		p = path.Dir(p)
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		err = os.MkdirAll(p, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
