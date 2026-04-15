package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	hdfs "github.com/colinmarc/hdfs/v2"
)

// HDFSProvider implements StorageProvider for HDFS using RPC protocol
type HDFSProvider struct {
	namenodeAddrs []string
	username      string
	client        *hdfs.Client
}

// NewHDFSProvider creates a new HDFS storage provider
// Configuration via environment variables:
//   - HDFS_NAMENODE_ADDRS: NameNode addresses (e.g., "localhost:9866")
//   - HDFS_USERNAME: HDFS user name (e.g., "hadoop")
func NewHDFSProvider() (StorageProvider, error) {
	addrs := os.Getenv("HDFS_NAMENODE_ADDRS")
	if addrs == "" {
		addrs = "localhost:8020"
	}

	username := os.Getenv("HDFS_USERNAME")
	if username == "" {
		username = "hadoop"
	}

	client, err := hdfs.NewClient(hdfs.ClientOptions{
		Addresses: strings.Split(addrs, ","),
		User:      username,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to HDFS: %w", err)
	}

	return &HDFSProvider{
		namenodeAddrs: strings.Split(addrs, ","),
		username:      username,
		client:        client,
	}, nil
}

// Scheme returns "hdfs"
func (p *HDFSProvider) Scheme() string {
	return "hdfs"
}

// Read reads a file from HDFS
func (p *HDFSProvider) Read(ctx context.Context, hdfsPath string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Normalize path - ensure starts with /
	hdfsPath = normalizeHDFSPath(hdfsPath)

	reader, err := p.client.Open(hdfsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open HDFS file %s: %w", hdfsPath, err)
	}

	return reader, nil
}

// Write writes data to an HDFS file
func (p *HDFSProvider) Write(ctx context.Context, hdfsPath string, data io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Normalize path - ensure starts with /
	hdfsPath = normalizeHDFSPath(hdfsPath)

	// Ensure parent directory exists
	parentDir := path.Dir(hdfsPath)
	if parentDir != "/" {
		if err := p.client.MkdirAll(parentDir, 0755); err != nil {
			// Ignore error if directory already exists
			_, statErr := p.client.Stat(parentDir)
			if statErr != nil && !os.IsNotExist(statErr) {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
		}
	}

	// Create the file
	writer, err := p.client.Create(hdfsPath)
	if err != nil {
		return fmt.Errorf("failed to create HDFS file: %w", err)
	}
	defer writer.Close()

	// Copy data to the file
	_, err = io.Copy(writer, data)
	if err != nil {
		return fmt.Errorf("failed to write to HDFS file: %w", err)
	}

	return nil
}

// Delete removes a file from HDFS
func (p *HDFSProvider) Delete(ctx context.Context, hdfsPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Normalize path - ensure starts with /
	hdfsPath = normalizeHDFSPath(hdfsPath)

	if err := p.client.Remove(hdfsPath); err != nil {
		return fmt.Errorf("failed to delete HDFS file: %w", err)
	}

	return nil
}

// List lists files in an HDFS directory
func (p *HDFSProvider) List(ctx context.Context, hdfsPath string) ([]FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Normalize path - ensure starts with /
	hdfsPath = normalizeHDFSPath(hdfsPath)

	if hdfsPath != "/" && !strings.HasSuffix(hdfsPath, "/") {
		hdfsPath = hdfsPath + "/"
	}

	fileInfos, err := p.client.ReadDir(hdfsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list HDFS directory: %w", err)
	}

	result := make([]FileInfo, 0, len(fileInfos))
	for _, fi := range fileInfos {
		result = append(result, FileInfo{
			Name:         fi.Name(),
			Path:         path.Join(hdfsPath, fi.Name()),
			IsDir:        fi.IsDir(),
			Size:         fi.Size(),
			LastModified: fi.ModTime(),
		})
	}

	return result, nil
}

// Stat returns metadata about an HDFS file
func (p *HDFSProvider) Stat(ctx context.Context, hdfsPath string) (*FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Normalize path - ensure starts with /
	hdfsPath = normalizeHDFSPath(hdfsPath)

	fi, err := p.client.Stat(hdfsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat HDFS file: %w", err)
	}

	return &FileInfo{
		Name:         fi.Name(),
		Path:         hdfsPath,
		IsDir:        fi.IsDir(),
		Size:         fi.Size(),
		LastModified: fi.ModTime(),
	}, nil
}

// Exists checks if a file exists in HDFS
func (p *HDFSProvider) Exists(ctx context.Context, hdfsPath string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	// Normalize path - ensure starts with /
	hdfsPath = normalizeHDFSPath(hdfsPath)

	_, err := p.client.Stat(hdfsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return true, nil
}

// Copy copies a file within HDFS
func (p *HDFSProvider) Copy(ctx context.Context, srcPath, dstPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Normalize paths
	srcPath = normalizeHDFSPath(srcPath)
	dstPath = normalizeHDFSPath(dstPath)

	// Read source
	reader, err := p.client.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer reader.Close()

	// Write to destination
	writer, err := p.client.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// normalizeHDFSPath ensures the path starts with /
func normalizeHDFSPath(p string) string {
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

// Close closes the HDFS client connection
func (p *HDFSProvider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// ConfigInfo returns the provider configuration for comparison
func (p *HDFSProvider) ConfigInfo() ProviderConfigInfo {
	return ProviderConfigInfo{
		Scheme: "hdfs",
	}
}

func init() {
	// Register HDFS provider
	Register("hdfs", NewHDFSProvider)
}