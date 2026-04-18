package coverage

const currentSchemaVersion = 1

type Ratio struct {
	Covered    int64 `json:"covered"`
	Statements int64 `json:"statements"`
}

func (r Ratio) Percent() float64 {
	if r.Statements == 0 {
		return 0
	}
	return (float64(r.Covered) / float64(r.Statements)) * 100
}

func (r Ratio) lessThan(other Ratio) bool {
	return r.Covered*other.Statements < other.Covered*r.Statements
}

type Snapshot struct {
	Total    Ratio
	Packages map[string]Ratio
}

type Baseline struct {
	SchemaVersion int              `json:"schema_version"`
	ModulePath    string           `json:"module_path"`
	Total         Ratio            `json:"total"`
	Packages      map[string]Ratio `json:"packages"`
}

type Scope string

const (
	ScopeTotal   Scope = "total"
	ScopePackage Scope = "package"
)

type Regression struct {
	Scope    Scope
	Name     string
	Baseline Ratio
	Current  Ratio
}
