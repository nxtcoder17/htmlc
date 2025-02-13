package functions

import (
	"io/fs"
	"path/filepath"
)

func RecursiveLs(dir string, patterns []string) ([]string, error) {
	var listings []string

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	if err := filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			if len(patterns) == 0 {
				listings = append(listings, path)
			}

			for _, pattern := range patterns {
				matched, err := filepath.Match(pattern, filepath.Base(path))
				if err != nil {
					return err
				}

				if matched {
					listings = append(listings, path[len(absDir):])
				}
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return listings, nil
}
