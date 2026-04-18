package coverage

import (
	"encoding/json"
	"fmt"
	"os"
)

func NewBaseline(modulePath string, snapshot Snapshot) Baseline {
	return Baseline{
		SchemaVersion: currentSchemaVersion,
		ModulePath:    modulePath,
		Total:         snapshot.Total,
		Packages:      clonePackages(snapshot.Packages),
	}
}

func ReadBaseline(path string) (Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Baseline{}, fmt.Errorf("ReadBaseline_os.ReadFile: %w", err)
	}

	var baseline Baseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return Baseline{}, fmt.Errorf("ReadBaseline_json.Unmarshal: %w", err)
	}

	if baseline.Packages == nil {
		baseline.Packages = map[string]Ratio{}
	}

	return baseline, nil
}

func WriteBaseline(path string, baseline Baseline) error {
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("WriteBaseline_json.MarshalIndent: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("WriteBaseline_os.WriteFile: %w", err)
	}

	return nil
}

func clonePackages(packages map[string]Ratio) map[string]Ratio {
	if len(packages) == 0 {
		return map[string]Ratio{}
	}

	cloned := make(map[string]Ratio, len(packages))
	for pkg, ratio := range packages {
		cloned[pkg] = ratio
	}

	return cloned
}
