package provider

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/geekjourneyx/agent-fs/pkg/sandbox"
)

// FileProviderInterface provides file handle access for efficient operations
type FileProviderInterface interface {
	FileHandle(ctx context.Context, path string) (*os.File, error)
}

// FileProvider implements StorageProvider for local filesystem
type FileProvider struct{}

// NewFileProvider creates a new local file provider
func NewFileProvider() (StorageProvider, error) {
	return &FileProvider{}, nil
}

// Scheme returns "file"
func (p *FileProvider) Scheme() string {
	return "file"
}

// Read reads a local file
func (p *FileProvider) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// FileHandle returns an *os.File for efficient file operations (like tail reading)
func (p *FileProvider) FileHandle(ctx context.Context, path string) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return os.Open(path)
}

// Write writes data to a local file
// Sandbox validation is performed to prevent path traversal attacks
func (p *FileProvider) Write(ctx context.Context, path string, data io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Sandbox validation for write operations
	resolvedPath, err := sandbox.ResolveWritePath(path)
	if err != nil {
		return err
	}
	// Ensure parent directory exists
	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(resolvedPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	return err
}

// Delete deletes a local file
// Sandbox validation is performed to prevent path traversal attacks
func (p *FileProvider) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Sandbox validation for delete operations (uses write validation since it modifies)
	resolvedPath, err := sandbox.ResolveWritePath(path)
	if err != nil {
		return err
	}
	return os.Remove(resolvedPath)
}

// List lists files in a local directory
func (p *FileProvider) List(ctx context.Context, path string) ([]FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	result := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, FileInfo{
			Name:         entry.Name(),
			Path:         filepath.Join(path, entry.Name()),
			IsDir:        entry.IsDir(),
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})
	}
	return result, nil
}

// Stat returns metadata about a local file
func (p *FileProvider) Stat(ctx context.Context, path string) (*FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &FileInfo{
		Name:         info.Name(),
		Path:         path,
		IsDir:        info.IsDir(),
		Size:         info.Size(),
		LastModified: info.ModTime(),
	}, nil
}

// Exists checks if a local file exists
func (p *FileProvider) Exists(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Copy copies a local file from source to destination
// Sandbox validation is performed on both source (read) and destination (write) paths
func (p *FileProvider) Copy(ctx context.Context, srcPath, dstPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Sandbox validation for source path (read operation)
	resolvedSrc, err := sandbox.ResolveReadPath(srcPath)
	if err != nil {
		return err
	}
	// Sandbox validation for destination path (write operation)
	resolvedDst, err := sandbox.ResolveWritePath(dstPath)
	if err != nil {
		return err
	}
	srcInfo, err := os.Stat(resolvedSrc)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDirectory(resolvedSrc, resolvedDst)
	}

	return copyFile(resolvedSrc, resolvedDst, srcInfo.Mode())
}

// directoryMode is the permission mode for created directories
const directoryMode = 0750

// copyFile copies a single file
func copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, directoryMode); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve permissions
	return os.Chmod(dst, mode)
}

// copyDirectory copies a directory recursively
func copyDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, directoryMode); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := copyFile(srcPath, dstPath, info.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	// Register the file provider
	Register("file", NewFileProvider)
}
