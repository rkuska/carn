package codex

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func listJSONLPaths(root string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !isJSONLExt(path) {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("filepath.WalkDir: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func scanRolloutsParallel(ctx context.Context, paths []string) ([]scannedRollout, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	results := make([]scannedRollout, len(paths))
	valid := make([]bool, len(paths))
	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	group, groupCtx := errgroup.WithContext(ctx)

	for i := range paths {
		index := i
		path := paths[i]

		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire_%s: %w", filepath.Base(path), err)
			}
			defer sem.Release(1)

			rollout, ok, err := scanRollout(path)
			if err != nil {
				return fmt.Errorf("scanRollout_%s: %w", filepath.Base(path), err)
			}
			if ok {
				results[index] = rollout
				valid[index] = true
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup.Wait: %w", err)
	}

	rollouts := make([]scannedRollout, 0, len(paths))
	for i, ok := range valid {
		if ok {
			rollouts = append(rollouts, results[i])
		}
	}
	return rollouts, nil
}
