package codex

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	src "github.com/rkuska/carn/internal/source"
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

func scanRolloutsParallel(
	ctx context.Context,
	paths []string,
) ([]scannedRollout, src.DriftReport, src.MalformedDataReport, error) {
	if len(paths) == 0 {
		return nil, src.DriftReport{}, src.MalformedDataReport{}, nil
	}

	results := make([]scannedRollout, len(paths))
	valid := make([]bool, len(paths))
	malformedValues := make([]string, len(paths))
	sem := semaphore.NewWeighted(int64(codexScanParallelism(len(paths))))
	group, groupCtx := errgroup.WithContext(ctx)
	log := zerolog.Ctx(ctx)

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
				if errors.Is(err, src.ErrMalformedRawData) {
					malformedValues[index] = path
					log.Debug().Err(err).Msgf("skipping %s", path)
					return nil
				}
				return fmt.Errorf("scanRollout_%s: %w", filepath.Base(path), err)
			}
			results[index] = rollout
			valid[index] = ok
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, src.DriftReport{}, src.MalformedDataReport{}, fmt.Errorf("errgroup.Wait: %w", err)
	}

	rollouts := make([]scannedRollout, 0, len(paths))
	drift := src.NewDriftReport()
	malformedData := src.NewMalformedDataReport()
	for i, ok := range valid {
		drift.Merge(results[i].drift)
		if ok {
			rollouts = append(rollouts, results[i])
		}
		malformedData.Record(malformedValues[i])
	}
	return rollouts, drift, malformedData, nil
}
