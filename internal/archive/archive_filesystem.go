package archive

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func fileNeedsSync(srcInfo os.FileInfo, dstPath string) bool {
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		return true
	}
	if srcInfo.Size() != dstInfo.Size() {
		return true
	}
	return srcInfo.ModTime().After(dstInfo.ModTime())
}

func statDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("srcFile.Stat: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}
	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("dstFile.Close: %w", err)
	}
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("os.Chtimes: %w", err)
	}
	return nil
}
