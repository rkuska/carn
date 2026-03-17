package archive

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func deferClose(c io.Closer, name string, retErr *error) {
	if closeErr := c.Close(); closeErr != nil && *retErr == nil {
		*retErr = fmt.Errorf("%s: %w", name, closeErr)
	}
}

func copyFile(src, dst string) (retErr error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer deferClose(srcFile, "srcFile.Close", &retErr)

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("srcFile.Stat: %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			deferClose(dstFile, "dstFile.Close", &retErr)
		}
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}
	if err = dstFile.Close(); err != nil {
		return fmt.Errorf("dstFile.Close: %w", err)
	}
	closed = true
	if err = os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("os.Chtimes: %w", err)
	}
	return nil
}
