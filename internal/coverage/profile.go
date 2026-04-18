package coverage

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

func ParseSnapshot(r io.Reader) (Snapshot, error) {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return Snapshot{}, fmt.Errorf("ParseSnapshot_scanner: %w", err)
		}
		return Snapshot{}, fmt.Errorf("ParseSnapshot: missing coverprofile header")
	}

	if !strings.HasPrefix(scanner.Text(), "mode: ") {
		return Snapshot{}, fmt.Errorf("ParseSnapshot: missing coverprofile header")
	}

	blocks := make(map[string]block)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parsedBlock, err := parseBlock(line)
		if err != nil {
			return Snapshot{}, fmt.Errorf("ParseSnapshot_parseBlock: %w", err)
		}

		current, ok := blocks[parsedBlock.key]
		if ok {
			current.covered = max(current.covered, parsedBlock.covered)
			blocks[parsedBlock.key] = current
			continue
		}
		blocks[parsedBlock.key] = parsedBlock
	}

	if err := scanner.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("ParseSnapshot_scanner: %w", err)
	}

	return collectSnapshot(blocks), nil
}

type block struct {
	key         string
	packagePath string
	statements  int64
	covered     int64
}

func parseBlock(line string) (block, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return block{}, fmt.Errorf("parseBlock: unexpected field count")
	}

	filePath, err := parseFilePath(fields[0])
	if err != nil {
		return block{}, fmt.Errorf("parseBlock_parseFilePath: %w", err)
	}

	statements, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return block{}, fmt.Errorf("parseBlock_parseStatements: %w", err)
	}

	count, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return block{}, fmt.Errorf("parseBlock_parseCount: %w", err)
	}

	covered := int64(0)
	if count > 0 {
		covered = statements
	}

	return block{
		key:         fields[0],
		packagePath: path.Dir(filePath),
		statements:  statements,
		covered:     covered,
	}, nil
}

func parseFilePath(field string) (string, error) {
	colon := strings.LastIndex(field, ":")
	if colon <= 0 {
		return "", fmt.Errorf("parseFilePath: missing location separator")
	}

	filePath := field[:colon]
	if path.Dir(filePath) == "." {
		return "", fmt.Errorf("parseFilePath: missing package path")
	}

	return filePath, nil
}

func collectSnapshot(blocks map[string]block) Snapshot {
	snapshot := Snapshot{
		Packages: make(map[string]Ratio, len(blocks)),
	}

	for _, block := range blocks {
		snapshot.Total.Statements += block.statements
		snapshot.Total.Covered += block.covered

		pkg := snapshot.Packages[block.packagePath]
		pkg.Statements += block.statements
		pkg.Covered += block.covered
		snapshot.Packages[block.packagePath] = pkg
	}

	return snapshot
}
