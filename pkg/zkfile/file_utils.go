package zkfile

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileInfo represents basic file information
type FileInfo struct {
	Size int64
	Path string
	Name string
}

// GetFileInfo returns file information
func GetFileInfo(path string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, NewIOError("failed to stat file").WithError(err).WithContext("path", path)
	}

	return &FileInfo{Path: path, Name: filepath.Base(path), Size: info.Size()}, nil
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir ensures that a directory exists
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewIOError("failed to create directory").
			WithError(err).
			WithContext("dir", dir)
	}
	return nil
}

// RemoveFile removes a file
func RemoveFile(path string) error {
	if err := os.Remove(path); err != nil {
		return NewIOError("failed to remove file").WithError(err).WithContext("path", path)
	}
	return nil
}

// RemoveDir removes a directory and all its contents
func RemoveDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return NewIOError("failed to remove directory").WithError(err).WithContext("dir", dir)
	}
	return nil
}

// GetDirSize calculates the total size of all files in a directory
func GetDirSize(dir string) (int64, error) {
	var size int64

	err := filepath.Walk(
		dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				size += info.Size()
			}
			return nil
		})

	if err != nil {
		return 0, NewIOError("failed to calculate directory size").WithError(err).WithContext("dir", dir)
	}

	return size, nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return NewIOError("failed to open source file").WithError(err).WithContext("src", src)
	}
	defer func() { _ = srcFile.Close() }()

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err = os.MkdirAll(dstDir, 0755); err != nil {
		return NewIOError("failed to create directory").WithError(err).WithContext("dir", dstDir)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return NewIOError("failed to create destination file").WithError(err).WithContext("dst", dst)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return NewIOError("failed to copy file").WithError(err).WithContext("src", src).WithContext("dst", dst)
	}

	if err = dstFile.Sync(); err != nil {
		return NewIOError("failed to sync file").WithError(err).WithContext("dst", dst)
	}

	// Copy file permissions
	srcInfo, err := srcFile.Stat()
	if err == nil {
		return os.Chmod(dst, srcInfo.Mode())
	}

	return nil
}

// calculateFileChecksum calculates SHA256 checksum of a file
func calculateFileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", NewIOError("failed to open file for checksum").WithError(err).WithContext("path", path)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", NewIOError("failed to calculate checksum").WithError(err).WithContext("path", path)
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
