package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhonxue/agent-fs/pkg/apperr"
)

const (
	// MaxDecompressedSize is the maximum allowed size for decompressed files (1GB)
	MaxDecompressedSize uint64 = 1 << 30
)

func Zip(sourcePath, outputZip string) (int64, error) {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return 0, apperr.Wrap(`local_zip`, apperr.CodeNotFound, `source path does not exist`, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputZip), 0o755); err != nil {
		return 0, apperr.Wrap(`local_zip`, apperr.CodeArchive, `failed to create output directory`, err)
	}

	outFile, err := os.Create(outputZip)
	if err != nil {
		return 0, apperr.Wrap(`local_zip`, apperr.CodeArchive, `failed to create zip file`, err)
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	rootParent := filepath.Dir(sourcePath)
	if !sourceInfo.IsDir() {
		rootParent = filepath.Dir(sourcePath)
	}

	var totalBytes int64
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, relErr := filepath.Rel(rootParent, path)
		if relErr != nil {
			return relErr
		}
		zipName := filepath.ToSlash(relPath)
		if info.IsDir() {
			if zipName == `.` {
				return nil
			}
			_, createErr := zw.Create(strings.TrimSuffix(zipName, `/`) + `/`)
			return createErr
		}

		header, headerErr := zip.FileInfoHeader(info)
		if headerErr != nil {
			return headerErr
		}
		header.Name = zipName
		header.Method = zip.Deflate

		writer, createErr := zw.CreateHeader(header)
		if createErr != nil {
			return createErr
		}

		file, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		n, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		totalBytes += n
		if closeErr != nil {
			return closeErr
		}
		return copyErr
	})
	if err != nil {
		return 0, apperr.Wrap(`local_zip`, apperr.CodeArchive, `failed to create zip archive`, err)
	}

	return totalBytes, nil
}

func Unzip(zipPath, destDir string) (int, int64, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `failed to open zip file`, err)
	}
	defer reader.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `failed to create destination directory`, err)
	}

	cleanDest, err := filepath.Abs(filepath.Clean(destDir))
	if err != nil {
		return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `invalid destination path`, err)
	}

	var (
		fileCount int
		totalSize int64
	)

	for _, f := range reader.File {
		targetPath := filepath.Join(cleanDest, filepath.Clean(f.Name))
		if !isSubPath(cleanDest, targetPath) {
			return 0, 0, apperr.New(`local_unzip`, apperr.CodePathTraversal, `zip contains unsafe path`)
		}

		if f.FileInfo().IsDir() {
			if mkErr := os.MkdirAll(targetPath, 0o755); mkErr != nil {
				return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `failed to create directory`, mkErr)
			}
			continue
		}

		if mkErr := os.MkdirAll(filepath.Dir(targetPath), 0o755); mkErr != nil {
			return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `failed to create parent directory`, mkErr)
		}

		src, openErr := f.Open()
		if openErr != nil {
			return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `failed to read zip entry`, openErr)
		}

		// Check uncompressed size to prevent decompression bomb attacks
		if f.UncompressedSize64 > MaxDecompressedSize {
			src.Close()
			return 0, 0, apperr.New(`local_unzip`, apperr.CodeArchive, `decompressed file exceeds maximum allowed size`)
		}

		dst, createErr := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if createErr != nil {
			src.Close()
			return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `failed to create destination file`, createErr)
		}

		// Use LimitedReader to prevent decompression bomb attacks
		limitedSrc := &io.LimitedReader{R: src, N: int64(MaxDecompressedSize)}
		n, copyErr := io.Copy(dst, limitedSrc)
		closeErr := dst.Close()
		srcCloseErr := src.Close()
		if copyErr != nil {
			return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `failed to extract zip entry`, copyErr)
		}
		if limitedSrc.N <= 0 {
			return 0, 0, apperr.New(`local_unzip`, apperr.CodeArchive, `decompressed file exceeded maximum allowed size`)
		}
		if closeErr != nil {
			return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `failed to finalize destination file`, closeErr)
		}
		if srcCloseErr != nil {
			return 0, 0, apperr.Wrap(`local_unzip`, apperr.CodeArchive, `failed to close zip entry stream`, srcCloseErr)
		}

		fileCount++
		totalSize += n
	}

	return fileCount, totalSize, nil
}

func isSubPath(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	if base == target {
		return true
	}
	return strings.HasPrefix(target, base+string(filepath.Separator))
}
