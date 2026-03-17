package claude

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	src "github.com/rkuska/carn/internal/source"
)

type projectFileClassifier struct {
	sourceDir     string
	rawDir        string
	dirName       string
	destInfoByRel map[string]fs.FileInfo
}

func newProjectFileClassifier(sourceDir, rawDir, dirName string) (projectFileClassifier, error) {
	destInfoByRel, err := buildProjectDestInfoIndex(filepath.Join(rawDir, dirName), rawDir)
	if err != nil {
		return projectFileClassifier{}, fmt.Errorf("buildProjectDestInfoIndex: %w", err)
	}
	return projectFileClassifier{
		sourceDir:     sourceDir,
		rawDir:        rawDir,
		dirName:       dirName,
		destInfoByRel: destInfoByRel,
	}, nil
}

func buildProjectDestInfoIndex(projectRawDir, rawDir string) (map[string]fs.FileInfo, error) {
	if _, err := os.Stat(projectRawDir); err != nil {
		if os.IsNotExist(err) {
			return make(map[string]fs.FileInfo), nil
		}
		return nil, fmt.Errorf("os.Stat: %w", err)
	}

	index := make(map[string]fs.FileInfo)
	err := filepath.WalkDir(projectRawDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("d.Info: %w", err)
		}
		rel, err := filepath.Rel(rawDir, path)
		if err != nil {
			return fmt.Errorf("filepath.Rel: %w", err)
		}
		index[rel] = info
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("filepath.WalkDir: %w", err)
	}
	return index, nil
}

func (c projectFileClassifier) classify(file sessionFile) (classifiedFile, bool) {
	slug, srcInfo, err := readSessionSlugAndInfo(file.path)
	if err != nil {
		return classifiedFile{}, false
	}

	gk := groupKey{dirName: c.dirName, slug: slug}
	if file.isSubagent || slug == "" {
		gk.slug = file.path
	}

	relPath := file.relPath
	if relPath == "" {
		relPath, err = filepath.Rel(c.sourceDir, file.path)
		if err != nil {
			return classifiedFile{}, false
		}
	}
	dstPath := filepath.Join(c.rawDir, relPath)

	dstInfo, dstExists := c.destInfoByRel[relPath]
	needsSync := !dstExists || src.FileNeedsSyncInfo(srcInfo, dstInfo)

	return classifiedFile{
		gk:        gk,
		needsSync: needsSync,
		dstExists: dstExists,
		srcPath:   file.path,
		dstPath:   dstPath,
	}, true
}

func readSessionSlugAndInfo(filePath string) (string, fs.FileInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return "", nil, fmt.Errorf("file.Stat: %w", err)
	}

	br := slugReaderPool.Get().(*bufio.Reader)
	br.Reset(file)
	defer slugReaderPool.Put(br)

	slug, err := extractSessionSlugFromReader(br)
	if err != nil {
		return "", nil, fmt.Errorf("extractSessionSlugFromReader: %w", err)
	}
	return slug, info, nil
}

func extractSessionSlugFromReader(br *bufio.Reader) (string, error) {
	var overflow []byte
	for {
		line, nextOverflow, err := readJSONLLine(br, overflow)
		overflow = nextOverflow

		if len(line) > 0 && extractType(line) == "user" {
			if slug := extractSlugFast(line); slug != "" {
				return slug, nil
			}
		}

		if err != nil {
			if err == io.EOF {
				return "", nil
			}
			return "", fmt.Errorf("readJSONLLine: %w", err)
		}
	}
}
