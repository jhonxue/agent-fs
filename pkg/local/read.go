package local

import (
	"bufio"
	"bytes"
	"io"
	"os"

	"github.com/jhonxue/agent-fs/pkg/apperr"
)

type ReadOptions struct {
	HeadLines int64
	TailLines int64
	Bytes     int64
}

type ReadResult struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	LineCount int    `json:"line_count"`
	ByteCount int    `json:"byte_count"`
	Truncated bool   `json:"truncated"`  // True if content was sliced
	SliceType string `json:"slice_type"` // "head", "tail", "bytes", or "full"
}

const (
	maxDefaultBytes = 10 * 1024 * 1024 // 10MB default limit for full reads
)

// ReadFile reads a file with optional slicing options.
// Only one of HeadLines, TailLines, or Bytes should be specified.
// If none are specified, the entire file is read (up to maxDefaultBytes).
func ReadFile(path string, opts ReadOptions) (ReadResult, error) {
	if opts.HeadLines > 0 && opts.TailLines > 0 {
		return ReadResult{}, apperr.New(`local_read`, apperr.CodeInvalidArg, `only one of --head or --tail can be specified`)
	}
	if opts.HeadLines > 0 && opts.Bytes > 0 {
		return ReadResult{}, apperr.New(`local_read`, apperr.CodeInvalidArg, `only one of --head or --bytes can be specified`)
	}
	if opts.TailLines > 0 && opts.Bytes > 0 {
		return ReadResult{}, apperr.New(`local_read`, apperr.CodeInvalidArg, `only one of --tail or --bytes can be specified`)
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ReadResult{}, apperr.Wrap(`local_read`, apperr.CodeNotFound, `file does not exist`, err)
		}
		return ReadResult{}, apperr.Wrap(`local_read`, apperr.CodeInternal, `failed to open file`, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return ReadResult{}, apperr.Wrap(`local_read`, apperr.CodeInternal, `failed to stat file`, err)
	}

	if info.IsDir() {
		return ReadResult{}, apperr.New(`local_read`, apperr.CodeInvalidArg, `path is a directory, not a file`)
	}

	switch {
	case opts.HeadLines > 0:
		return readHeadLines(file, path, opts.HeadLines, info.Size())
	case opts.TailLines > 0:
		return readTailLines(file, path, opts.TailLines, info.Size())
	case opts.Bytes > 0:
		return readBytes(file, path, opts.Bytes)
	default:
		return readFull(file, path, info.Size())
	}
}

func readHeadLines(file *os.File, path string, n int64, fileSize int64) (ReadResult, error) {
	var content bytes.Buffer
	var lineCount int

	scanner := bufio.NewScanner(file)
	for scanner.Scan() && lineCount < int(n) {
		content.WriteString(scanner.Text())
		content.WriteByte('\n')
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return ReadResult{}, apperr.Wrap(`local_read`, apperr.CodeInternal, `failed to read file`, err)
	}

	truncated := lineCount == int(n) && !scannerEOF(file, fileSize)

	return ReadResult{
		Path:      path,
		Content:   content.String(),
		LineCount: lineCount,
		ByteCount: content.Len(),
		Truncated: truncated,
		SliceType: "head",
	}, nil
}

func readTailLines(file *os.File, path string, n int64, fileSize int64) (ReadResult, error) {
	// For tail, we need to read all lines and keep the last N
	// This is simpler but uses more memory for large files
	// For a more efficient implementation, we'd scan from the end

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return ReadResult{}, apperr.Wrap(`local_read`, apperr.CodeInternal, `failed to read file`, err)
	}

	start := 0
	if len(lines) > int(n) {
		start = len(lines) - int(n)
	}

	var content bytes.Buffer
	for i := start; i < len(lines); i++ {
		content.WriteString(lines[i])
		content.WriteByte('\n')
	}

	lineCount := len(lines) - start

	return ReadResult{
		Path:      path,
		Content:   content.String(),
		LineCount: lineCount,
		ByteCount: content.Len(),
		Truncated: start > 0,
		SliceType: "tail",
	}, nil
}

func readBytes(file *os.File, path string, n int64) (ReadResult, error) {
	buf := make([]byte, n)
	read, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return ReadResult{}, apperr.Wrap(`local_read`, apperr.CodeInternal, `failed to read file`, err)
	}

	// Only use the bytes that were actually read
	content := buf[:read]

	// Truncated is true if we read the full requested amount (file has more)
	// If we got ErrUnexpectedEOF, we reached the end of the file
	truncated := (err == nil || err == io.EOF)

	return ReadResult{
		Path:      path,
		Content:   string(content),
		LineCount: 0,
		ByteCount: read,
		Truncated: truncated,
		SliceType: "bytes",
	}, nil
}

func readFull(file *os.File, path string, fileSize int64) (ReadResult, error) {
	if fileSize > maxDefaultBytes {
		return ReadResult{}, apperr.New(`local_read`, apperr.CodeInvalidArg,
			`file too large for full read; use --head, --tail, or --bytes to read a portion`)
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return ReadResult{}, apperr.Wrap(`local_read`, apperr.CodeInternal, `failed to read file`, err)
	}

	lineCount := bytes.Count(content, []byte{'\n'})
	if len(content) > 0 && content[len(content)-1] != '\n' {
		lineCount++ // Last line without newline
	}

	return ReadResult{
		Path:      path,
		Content:   string(content),
		LineCount: lineCount,
		ByteCount: len(content),
		Truncated: false,
		SliceType: "full",
	}, nil
}

// scannerEOF checks if we've read the entire file
func scannerEOF(file *os.File, fileSize int64) bool {
	pos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return false
	}
	return pos >= fileSize
}
