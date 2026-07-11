// Package lockfile reads and writes poly.json, a project-local file
// listing packages to install with `poly install` (no arguments) --
// the same idea as package.json/requirements.txt, but adapter-agnostic.
package lockfile

import (
	"encoding/json"
	"os"
)

const FileName = "poly.json"

type File struct {
	Packages []string `json:"packages"`
}

// Load reads poly.json from the current directory. found is false if
// it doesn't exist (not an error).
func Load() (f *File, found bool, err error) {
	data, err := os.ReadFile(FileName)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	f = &File{}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, false, err
	}
	return f, true, nil
}

func Save(f *File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(FileName, data, 0o644)
}

// Exists reports whether poly.json is present in the current directory.
func Exists() bool {
	_, err := os.Stat(FileName)
	return err == nil
}
