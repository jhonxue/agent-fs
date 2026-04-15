package sandbox

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jhonxue/agent-fs/pkg/apperr"
)

const workspaceEnv = `AFS_WORKSPACE`

func ResolveReadPath(inputPath string) (string, error) {
	return resolvePath(inputPath)
}

func ResolveWritePath(inputPath string) (string, error) {
	return resolvePath(inputPath)
}

func resolvePath(inputPath string) (string, error) {
	if strings.TrimSpace(inputPath) == `` {
		return ``, apperr.New(`sandbox`, apperr.CodeInvalidArg, `path is required`)
	}

	absPath, err := filepath.Abs(filepath.Clean(inputPath))
	if err != nil {
		return ``, apperr.Wrap(`sandbox`, apperr.CodeInvalidArg, `failed to resolve path`, err)
	}

	root, err := workspaceRoot()
	if err != nil {
		return ``, err
	}
	if root == `` {
		return absPath, nil
	}

	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return ``, apperr.Wrap(`sandbox`, apperr.CodePathTraversal, `failed to validate sandbox path`, err)
	}
	if rel == `..` || strings.HasPrefix(rel, `..`+string(filepath.Separator)) {
		return ``, apperr.New(`sandbox`, apperr.CodePathTraversal, `access denied: path is outside AFS_WORKSPACE`)
	}
	return absPath, nil
}

func workspaceRoot() (string, error) {
	ws := strings.TrimSpace(os.Getenv(workspaceEnv))
	if ws == `` {
		return ``, nil
	}
	abs, err := filepath.Abs(filepath.Clean(ws))
	if err != nil {
		return ``, apperr.Wrap(`sandbox`, apperr.CodeInvalidArg, `invalid AFS_WORKSPACE path`, err)
	}
	return abs, nil
}
