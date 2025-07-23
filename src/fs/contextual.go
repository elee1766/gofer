package fs

import (
	"os"
	"path/filepath"

	"github.com/elee1766/gofer/src/shell"
	"github.com/spf13/afero"
)

// ContextualFs creates an afero.Fs that resolves paths relative to a working directory
type ContextualFs struct {
	afero.Fs
	workingDir string
}

// NewContextualFs creates a new ContextualFs with the given working directory
func NewContextualFs(baseFs afero.Fs, workingDir string) *ContextualFs {
	return &ContextualFs{
		Fs:         baseFs,
		workingDir: workingDir,
	}
}

// NewContextualFsFromShell creates a ContextualFs using the shell manager's working directory
func NewContextualFsFromShell(baseFs afero.Fs, shellManager *shell.ShellManager, conversationID string) (*ContextualFs, error) {
	workingDir, err := shellManager.GetCurrentDirectory(conversationID)
	if err != nil {
		return nil, err
	}
	
	// If no shell exists yet, use the base filesystem
	if workingDir == "" {
		return &ContextualFs{
			Fs:         baseFs,
			workingDir: "",
		}, nil
	}
	
	return NewContextualFs(baseFs, workingDir), nil
}

// resolvePath resolves a path relative to the working directory if it's not absolute
func (c *ContextualFs) resolvePath(path string) string {
	// Handle empty path as current directory
	if path == "" {
		if c.workingDir == "" {
			return "."
		}
		return c.workingDir
	}
	
	if filepath.IsAbs(path) || c.workingDir == "" {
		return path
	}
	return filepath.Join(c.workingDir, path)
}

// Override methods to resolve paths relative to working directory

func (c *ContextualFs) Open(name string) (afero.File, error) {
	return c.Fs.Open(c.resolvePath(name))
}

func (c *ContextualFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return c.Fs.OpenFile(c.resolvePath(name), flag, perm)
}

func (c *ContextualFs) Remove(name string) error {
	return c.Fs.Remove(c.resolvePath(name))
}

func (c *ContextualFs) RemoveAll(path string) error {
	return c.Fs.RemoveAll(c.resolvePath(path))
}

func (c *ContextualFs) Rename(oldname, newname string) error {
	return c.Fs.Rename(c.resolvePath(oldname), c.resolvePath(newname))
}

func (c *ContextualFs) Stat(name string) (os.FileInfo, error) {
	return c.Fs.Stat(c.resolvePath(name))
}

func (c *ContextualFs) Create(name string) (afero.File, error) {
	return c.Fs.Create(c.resolvePath(name))
}

func (c *ContextualFs) Mkdir(name string, perm os.FileMode) error {
	return c.Fs.Mkdir(c.resolvePath(name), perm)
}

func (c *ContextualFs) MkdirAll(path string, perm os.FileMode) error {
	return c.Fs.MkdirAll(c.resolvePath(path), perm)
}

// GetWorkingDir returns the current working directory
func (c *ContextualFs) GetWorkingDir() string {
	return c.workingDir
}