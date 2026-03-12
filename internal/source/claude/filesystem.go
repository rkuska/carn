package claude

import "os"

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
