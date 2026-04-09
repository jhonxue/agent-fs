//go:build cephfs && cgo
// +build cephfs,cgo

package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ceph/go-ceph/cephfs"
)

// CephFSProvider implements StorageProvider for CephFS using libcephfs API
type CephFSProvider struct {
	mount *cephfs.MountInfo
}

// NewCephFSProvider creates a new CephFS storage provider
// Configuration via environment variables:
//   - CEPHFS_MON_HOSTS: Ceph monitor addresses (e.g., "192.168.1.1:6789,192.168.1.2:6789")
//   - CEPHFS_AUTH_ID: Ceph user ID (e.g., "admin")
//   - CEPHFS_KEYRING_PATH: Path to keyring file
//   - CEPHFS_CONF_PATH: Path to ceph.conf file
//   - CEPHFS_SECRET: Ceph secret key (alternative to keyring)
func NewCephFSProvider() (StorageProvider, error) {
	// Initialize go-ceph library
	if err := cephfs.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize cephfs library: %w", err)
	}

	// Try to create from config file first
	confPath := os.Getenv("CEPHFS_CONF_PATH")
	if confPath == "" {
		// Try default locations
		defaultPaths := []string{
			"/etc/ceph/ceph.conf",
			"/etc/ceph/cephfs.conf",
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				confPath = p
				break
			}
		}
	}

	var mount *cephfs.MountInfo
	var err error

	if confPath != "" {
		// Try to create from config file
		mount, err = cephfs.CreateFromConfigFile(confPath, os.Getenv("CEPHFS_KEYRING_PATH"))
	} else {
		// Try to create from monitor hosts
		monHosts := os.Getenv("CEPHFS_MON_HOSTS")
		if monHosts == "" && confPath == "" {
			return nil, &providerConfigError{
				msg: "CephFS provider requires CEPHFS_CONF_PATH or CEPHFS_MON_HOSTS environment variable",
			}
		}
		authID := os.Getenv("CEPHFS_AUTH_ID")
		if authID == "" {
			authID = "admin"
		}
		keyringPath := os.Getenv("CEPHFS_KEYRING_PATH")
		secret := os.Getenv("CEPHFS_SECRET")

		mount, err = cephfs.CreateFromMonHost(strings.Split(monHosts, ","), authID, keyringPath, secret)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to CephFS: %w", err)
	}

	return &CephFSProvider{
		mount: mount,
	}, nil
}

// Scheme returns "cephfs"
func (p *CephFSProvider) Scheme() string {
	return "cephfs"
}

// Read reads a file from CephFS
func (p *CephFSProvider) Read(ctx context.Context, cephPath string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// CephFS paths should start with /
	cephPath = normalizePath(cephPath)

	file, err := p.mount.Open(cephPath, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open cephfs file %s: %w", cephPath, err)
	}

	return file, nil
}

// FileHandle returns a file handle for efficient operations
// Note: CephFS file handles cannot be directly converted to *os.File
// Use Read method instead for streaming file data
func (p *CephFSProvider) FileHandle(ctx context.Context, cephPath string) (*os.File, error) {
	return nil, &notImplementedError{"FileHandle not supported for CephFS, use Read method instead"}
}

// Write writes data to a CephFS file
func (p *CephFSProvider) Write(ctx context.Context, cephPath string, data io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cephPath = normalizePath(cephPath)

	// Ensure parent directory exists
	parentDir := path.Dir(cephPath)
	if parentDir != "/" {
		if err := p.mount.MkdirAll(parentDir, 0755); err != nil {
			// Ignore error if directory already exists
			stat, statErr := p.mount.Statx(parentDir)
			if statErr == nil && stat.Mode&cephfs.S_IFDIR == 0 {
				return fmt.Errorf("parent path is not a directory: %s", parentDir)
			}
		}
	}

	// Create or truncate the file
	fd, err := p.mount.Open(cephPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create cephfs file: %w", err)
	}
	defer fd.Close()

	// Copy data to the file
	_, err = io.Copy(fd, data)
	if err != nil {
		return fmt.Errorf("failed to write to cephfs file: %w", err)
	}

	return nil
}

// Delete removes a file from CephFS
func (p *CephFSProvider) Delete(ctx context.Context, cephPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cephPath = normalizePath(cephPath)

	err := p.mount.Unlink(cephPath)
	if err != nil {
		return fmt.Errorf("failed to delete cephfs file: %w", err)
	}

	return nil
}

// List lists files in a CephFS directory
func (p *CephFSProvider) List(ctx context.Context, cephPath string) ([]FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cephPath = normalizePath(cephPath)

	entries, err := p.mount.ReadDir(cephPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list cephfs directory: %w", err)
	}

	result := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		result = append(result, FileInfo{
			Name:         entry.Name(),
			Path:         path.Join(cephPath, entry.Name()),
			IsDir:        entry.IsDir(),
			Size:         entry.Size(),
			LastModified: time.Unix(entry.Mtim.Sec, int64(entry.Mtim.Nsec)),
		})
	}

	return result, nil
}

// Stat returns metadata about a CephFS file
func (p *CephFSProvider) Stat(ctx context.Context, cephPath string) (*FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cephPath = normalizePath(cephPath)

	stat, err := p.mount.Statx(cephPath, cephfs.StatxBasicStats)
	if err != nil {
		return nil, fmt.Errorf("failed to stat cephfs file: %w", err)
	}

	return &FileInfo{
		Name:         path.Base(cephPath),
		Path:         cephPath,
		IsDir:        stat.Mode&cephfs.S_IFDIR != 0,
		Size:         stat.Size,
		LastModified: time.Unix(stat.Mtim.Sec, int64(stat.Mtim.Nsec)),
	}, nil
}

// Exists checks if a file exists in CephFS
func (p *CephFSProvider) Exists(ctx context.Context, cephPath string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	cephPath = normalizePath(cephPath)

	_, err := p.mount.Statx(cephPath, cephfs.StatxBasicStats)
	if err != nil {
		// ENOENT means file doesn't exist
		if strings.Contains(err.Error(), "ENOENT") || err == cephfs.ErrNotExist {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return true, nil
}

// Copy copies a file within CephFS
func (p *CephFSProvider) Copy(ctx context.Context, srcPath, dstPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	srcPath = normalizePath(srcPath)
	dstPath = normalizePath(dstPath)

	// Try to use CopyFile if available
	err := p.mount.CopyFile(srcPath, dstPath)
	if err == nil {
		return nil
	}

	// Fallback: read source and write to destination
	srcFile, err := p.mount.Open(srcPath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Ensure parent directory exists
	parentDir := path.Dir(dstPath)
	if parentDir != "/" {
		_ = p.mount.MkdirAll(parentDir, 0755)
	}

	dstFile, err := p.mount.Open(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// normalizePath ensures the path starts with /
func normalizePath(p string) string {
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

func init() {
	// Register CephFS provider
	Register("cephfs", NewCephFSProvider)
}
