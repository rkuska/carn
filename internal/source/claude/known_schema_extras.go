package claude

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const knownSchemaExtraStatusIgnored = "ignored"

type knownSchemaExtraEntry struct {
	Status      string          `json:"status"`
	Path        string          `json:"path"`
	RecordTypes []string        `json:"record_types"`
	Description string          `json:"description"`
	FutureUse   string          `json:"future_use"`
	FirstSeen   string          `json:"first_seen"`
	Example     json.RawMessage `json:"example"`
}

type knownSchemaExtrasDocument struct {
	Provider   string                                      `json:"provider"`
	Categories map[string]map[string]knownSchemaExtraEntry `json:"categories"`
}

type knownSchemaExtrasCatalog struct {
	categories    map[string]map[string]knownSchemaExtraEntry
	rawCategories map[string]map[string]struct{}
}

type knownSchemaExtraPathSegment struct {
	name    string
	isArray bool
}

var knownSchemaExtraCategories = map[string]struct{}{
	"envelope_field":     {},
	"message_field":      {},
	"usage_field":        {},
	"record_type":        {},
	"content_block_type": {},
}

//go:embed known_schema_extras.json
var knownSchemaExtrasJSON []byte

var claudeKnownSchemaExtras = mustLoadKnownSchemaExtras()

func mustLoadKnownSchemaExtras() knownSchemaExtrasCatalog {
	var doc knownSchemaExtrasDocument
	if err := json.Unmarshal(knownSchemaExtrasJSON, &doc); err != nil {
		panic(fmt.Errorf("mustLoadKnownSchemaExtras_jsonUnmarshal: %w", err))
	}
	if err := validateKnownSchemaExtrasDocument(doc); err != nil {
		panic(fmt.Errorf("mustLoadKnownSchemaExtras_validateKnownSchemaExtrasDocument: %w", err))
	}
	return knownSchemaExtrasCatalog{
		categories:    doc.Categories,
		rawCategories: buildKnownSchemaExtraRawCategories(doc.Categories),
	}
}

func buildKnownSchemaExtraRawCategories(
	categories map[string]map[string]knownSchemaExtraEntry,
) map[string]map[string]struct{} {
	rawCategories := make(map[string]map[string]struct{}, len(categories))
	for category, values := range categories {
		rawValues := make(map[string]struct{}, len(values))
		for value := range values {
			rawValues[strconv.Quote(value)] = struct{}{}
		}
		rawCategories[category] = rawValues
	}
	return rawCategories
}

func validateKnownSchemaExtrasDocument(doc knownSchemaExtrasDocument) error {
	if doc.Provider != "claude" {
		return fmt.Errorf("provider must be claude")
	}
	if len(doc.Categories) == 0 {
		return fmt.Errorf("categories must not be empty")
	}
	for category, values := range doc.Categories {
		if _, ok := knownSchemaExtraCategories[category]; !ok {
			return fmt.Errorf("unknown category %q", category)
		}
		if len(values) == 0 {
			return fmt.Errorf("category %q must not be empty", category)
		}
		for value, entry := range values {
			if err := validateKnownSchemaExtraEntry(value, entry); err != nil {
				return fmt.Errorf("%s/%s: %w", category, value, err)
			}
		}
	}
	return nil
}

func validateKnownSchemaExtraEntry(value string, entry knownSchemaExtraEntry) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("value must not be empty")
	}
	if entry.Status != knownSchemaExtraStatusIgnored {
		return fmt.Errorf("status %q is not supported", entry.Status)
	}
	if err := validateKnownSchemaExtraDocumentation(entry); err != nil {
		return err
	}
	if err := validateKnownSchemaExtraRecordTypes(entry.RecordTypes); err != nil {
		return err
	}
	if err := validateKnownSchemaExtraFirstSeen(entry.FirstSeen); err != nil {
		return err
	}
	return validateKnownSchemaExtraExample(entry.Path, entry.Example)
}

func validateKnownSchemaExtraDocumentation(entry knownSchemaExtraEntry) error {
	if strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("path must not be empty")
	}
	if strings.TrimSpace(entry.Description) == "" {
		return fmt.Errorf("description must not be empty")
	}
	if strings.TrimSpace(entry.FutureUse) == "" {
		return fmt.Errorf("future_use must not be empty")
	}
	return nil
}

func validateKnownSchemaExtraRecordTypes(recordTypes []string) error {
	if len(recordTypes) == 0 {
		return fmt.Errorf("record_types must not be empty")
	}
	for _, recordType := range recordTypes {
		if strings.TrimSpace(recordType) == "" {
			return fmt.Errorf("record_types must not contain empty values")
		}
	}
	return nil
}

func validateKnownSchemaExtraFirstSeen(firstSeen string) error {
	if strings.TrimSpace(firstSeen) == "" {
		return fmt.Errorf("first_seen must not be empty")
	}
	if _, err := time.Parse(time.DateOnly, firstSeen); err != nil {
		return fmt.Errorf("time.Parse first_seen: %w", err)
	}
	return nil
}

func validateKnownSchemaExtraExample(path string, example json.RawMessage) error {
	if len(example) == 0 {
		return fmt.Errorf("example must not be empty")
	}
	var decoded any
	if err := json.Unmarshal(example, &decoded); err != nil {
		return fmt.Errorf("json.Unmarshal example: %w", err)
	}
	if !exampleContainsPath(decoded, path) {
		return fmt.Errorf("example does not contain path %q", path)
	}
	return nil
}

func (c knownSchemaExtrasCatalog) Has(category, value string) bool {
	values := c.categories[category]
	if len(values) == 0 {
		return false
	}
	_, ok := values[value]
	return ok
}

func (c knownSchemaExtrasCatalog) HasRaw(category string, raw []byte) bool {
	values := c.rawCategories[category]
	if len(values) == 0 {
		return false
	}
	return hasRawJSONString(raw, values)
}

func (c knownSchemaExtrasCatalog) Categories() map[string]map[string]knownSchemaExtraEntry {
	return c.categories
}

func exampleContainsPath(decoded any, path string) bool {
	return exampleContainsPathSegments(decoded, strings.Split(path, "."))
}

func exampleContainsPathSegments(current any, segments []string) bool {
	if len(segments) == 0 {
		return true
	}
	segment, ok := parseKnownSchemaExtraPathSegment(segments[0])
	if !ok {
		return false
	}
	if segment.isArray {
		return exampleContainsArrayPathSegment(current, segment.name, segments[1:])
	}
	return exampleContainsObjectPathSegment(current, segment.name, segments[1:])
}

func parseKnownSchemaExtraPathSegment(raw string) (knownSchemaExtraPathSegment, bool) {
	trimmed := strings.TrimSpace(raw)
	name := strings.TrimSuffix(trimmed, "[]")
	if name == "" {
		return knownSchemaExtraPathSegment{}, false
	}
	return knownSchemaExtraPathSegment{
		name:    name,
		isArray: strings.HasSuffix(trimmed, "[]"),
	}, true
}

func exampleContainsArrayPathSegment(current any, name string, tail []string) bool {
	items, ok := lookupKnownSchemaExtraArrayField(current, name)
	if !ok || len(items) == 0 {
		return false
	}
	for _, item := range items {
		if exampleContainsPathSegments(item, tail) {
			return true
		}
	}
	return false
}

func exampleContainsObjectPathSegment(current any, name string, tail []string) bool {
	next, ok := lookupKnownSchemaExtraObjectField(current, name)
	if !ok {
		return false
	}
	return exampleContainsPathSegments(next, tail)
}

func lookupKnownSchemaExtraArrayField(current any, name string) ([]any, bool) {
	fields, ok := current.(map[string]any)
	if !ok {
		return nil, false
	}
	value, ok := fields[name]
	if !ok {
		return nil, false
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	return items, true
}

func lookupKnownSchemaExtraObjectField(current any, name string) (any, bool) {
	fields, ok := current.(map[string]any)
	if !ok {
		return nil, false
	}
	value, ok := fields[name]
	if !ok {
		return nil, false
	}
	return value, true
}
