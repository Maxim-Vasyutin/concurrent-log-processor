package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ScanError struct {
	Path  string
	Error error
}

type DirectoryScanResult struct {
	RootPath   string
	LogFiles   []string
	ScanErrors []ScanError
}

type LogFileScanner struct{}

func (s LogFileScanner) Scan(rootPath string) (DirectoryScanResult, error) {
	return ScanLogDirectory(rootPath)
}

func ValidateDirectory(rootPath string) error {
	if strings.TrimSpace(rootPath) == "" {
		return fmt.Errorf("root path is required")
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		return fmt.Errorf("directory %q: %w", rootPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q is not a directory", rootPath)
	}
	return nil
}

func ScanLogDirectory(rootPath string) (DirectoryScanResult, error) {
	if err := ValidateDirectory(rootPath); err != nil {
		return DirectoryScanResult{}, err
	}

	result := DirectoryScanResult{RootPath: rootPath}
	walkErr := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			result.ScanErrors = append(result.ScanErrors, ScanError{Path: path, Error: err})
			return nil
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".log" {
			return nil
		}

		result.LogFiles = append(result.LogFiles, path)
		return nil
	})
	if walkErr != nil {
		return DirectoryScanResult{}, fmt.Errorf("scan root %q: %w", rootPath, walkErr)
	}

	sort.Strings(result.LogFiles)
	return result, nil
}
