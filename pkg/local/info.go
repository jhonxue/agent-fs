package local

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jhonxue/agent-fs/pkg/apperr"
)

type FileInfo struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Type         string `json:"type"` // "file" or "directory"
	SizeBytes    int64  `json:"size_bytes"`
	Mode         string `json:"mode"`          // Unix permissions in octal
	ModifiedTime string `json:"modified_time"` // RFC3339 format
	IsDir        bool   `json:"is_dir"`
}

type DirectoryInfo struct {
	Path         string `json:"path"`
	FileCount    int    `json:"file_count"`
	TotalBytes   int64  `json:"total_bytes"`
	ModifiedTime string `json:"modified_time"`
	Mode         string `json:"mode"`
}

// GetInfo retrieves metadata for a file or directory.
// For files, it returns FileInfo with detailed metadata.
// For directories, it returns DirectoryInfo with aggregated statistics.
func GetInfo(path string, includeDirDetails bool) (FileInfo, DirectoryInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FileInfo{}, DirectoryInfo{}, apperr.Wrap(`local_info`, apperr.CodeNotFound, `path does not exist`, err)
		}
		return FileInfo{}, DirectoryInfo{}, apperr.Wrap(`local_info`, apperr.CodeInvalidArg, `failed to stat path`, err)
	}

	modeStr := fmt.Sprintf("%04o", info.Mode().Perm())
	modTime := info.ModTime().Format(time.RFC3339)

	name := filepath.Base(path)
	if path == "/" || path == "." {
		name = path
	}

	if info.IsDir() {
		dirInfo := DirectoryInfo{
			Path:         path,
			FileCount:    0,
			TotalBytes:   0,
			ModifiedTime: modTime,
			Mode:         modeStr,
		}

		if includeDirDetails {
			fileCount, totalBytes, err := walkDirectory(path)
			if err != nil {
				return FileInfo{}, DirectoryInfo{}, err
			}
			dirInfo.FileCount = fileCount
			dirInfo.TotalBytes = totalBytes
		}

		return FileInfo{
			Name:         name,
			Path:         path,
			Type:         "directory",
			SizeBytes:    0,
			Mode:         modeStr,
			ModifiedTime: modTime,
			IsDir:        true,
		}, dirInfo, nil
	}

	return FileInfo{
		Name:         name,
		Path:         path,
		Type:         "file",
		SizeBytes:    info.Size(),
		Mode:         modeStr,
		ModifiedTime: modTime,
		IsDir:        false,
	}, DirectoryInfo{}, nil
}

// walkDirectory walks a directory and returns file count and total bytes.
func walkDirectory(dirPath string) (int, int64, error) {
	var fileCount int
	var totalBytes int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
			totalBytes += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, 0, apperr.Wrap(`local_info`, apperr.CodeInternal, `failed to walk directory`, err)
	}

	return fileCount, totalBytes, nil
}
