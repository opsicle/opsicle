package common

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type MatchedResource struct {
	Data   []byte
	Header Resource
	Path   string
}

// FindResourceInFilesystem returns all YAML files under rootDir where:
// - .Metadata.Name == targetName
// - .Type == resourceType
//
// Returns an empty slice if no results are found. Only throws an error when
// a logical error is emitted
func FindResourceInFilesystem(rootDir, targetName, resourceType string) ([]MatchedResource, error) {
	var matches []MatchedResource

	if err := filepath.WalkDir(rootDir, func(filePath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("unknown error while walking directory[%s]: %s", rootDir, err)
		}

		if dirEntry.IsDir() ||
			!(strings.HasSuffix(dirEntry.Name(), ".yaml") || strings.HasSuffix(dirEntry.Name(), ".yml")) {
			return nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file at path[%s]: %s", filePath, err)
		}

		var resourceHeader Resource
		if err := yaml.Unmarshal(data, &resourceHeader); err != nil {
			return nil // skip invalid YAML
		}

		if resourceHeader.Metadata.Name == targetName && resourceHeader.Type == resourceType {
			matches = append(matches, MatchedResource{
				Data:   data,
				Header: resourceHeader,
				Path:   filePath,
			})
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return matches, nil
}
