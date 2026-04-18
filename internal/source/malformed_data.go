package source

import (
	"errors"
	"slices"
	"strings"
)

var ErrMalformedRawData = errors.New("malformed raw data")

type malformedRawDataError struct {
	err error
}

func (e malformedRawDataError) Error() string {
	return e.err.Error()
}

func (e malformedRawDataError) Unwrap() error {
	return e.err
}

func (e malformedRawDataError) Is(target error) bool {
	return target == ErrMalformedRawData
}

func MarkMalformedRawData(err error) error {
	if err == nil || errors.Is(err, ErrMalformedRawData) {
		return err
	}
	return malformedRawDataError{err: err}
}

type MalformedDataReport struct {
	values map[string]struct{}
}

func NewMalformedDataReport() MalformedDataReport {
	return MalformedDataReport{
		values: make(map[string]struct{}),
	}
}

func (r *MalformedDataReport) Record(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if r.values == nil {
		r.values = make(map[string]struct{})
	}
	r.values[value] = struct{}{}
}

func (r *MalformedDataReport) Merge(other MalformedDataReport) {
	if len(other.values) == 0 {
		return
	}
	if r.values == nil {
		r.values = make(map[string]struct{}, len(other.values))
	}
	for value := range other.values {
		r.values[value] = struct{}{}
	}
}

func (r MalformedDataReport) Empty() bool {
	return len(r.values) == 0
}

func (r MalformedDataReport) Count() int {
	return len(r.values)
}

func (r MalformedDataReport) Values() []string {
	values := make([]string, 0, len(r.values))
	for value := range r.values {
		values = append(values, value)
	}
	slices.Sort(values)
	return values
}

type ProviderMalformedDataReports = ProviderReports[MalformedDataReport, *MalformedDataReport]

func NewProviderMalformedDataReports() ProviderMalformedDataReports {
	return NewProviderReports[MalformedDataReport]()
}
