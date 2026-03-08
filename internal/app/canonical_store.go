package app

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	storeSchemaVersion       = 1
	storeProjectionVersion   = 1
	storeSearchCorpusVersion = 1

	catalogMagic    = "CLDSCAT1"
	transcriptMagic = "CLDSSES1"
	searchMagic     = "CLDSSRH1"
)

type storeManifest struct {
	SchemaVersion       int                  `json:"schema_version"`
	ProjectionVersion   int                  `json:"projection_version"`
	SearchCorpusVersion int                  `json:"search_corpus_version"`
	Provider            conversationProvider `json:"provider"`
}

type searchUnit struct {
	conversationID string
	text           string
}

type searchCorpus struct {
	units []searchUnit
}

func (c searchCorpus) Len() int {
	return len(c.units)
}

func (c searchCorpus) String(i int) string {
	return c.units[i].text
}

func (r conversationRepository) searchCorpus(
	ctx context.Context,
	archiveDir string,
) (searchCorpus, error) {
	var merged searchCorpus
	for _, source := range r.sources {
		corpus, err := source.searchCorpus(ctx, archiveDir)
		if err != nil {
			return searchCorpus{}, fmt.Errorf("searchCorpus_%s: %w", source.provider(), err)
		}
		merged.units = append(merged.units, corpus.units...)
	}
	return merged, nil
}

func rebuildCanonicalStore(
	ctx context.Context,
	archiveDir string,
	provider conversationProvider,
	changedRawPaths []string,
) error {
	rawDir := providerRawDir(archiveDir, provider)
	if _, err := statDir(rawDir); err != nil {
		return fmt.Errorf("statDir_raw: %w", err)
	}

	sessions, err := scanSessions(ctx, rawDir)
	if err != nil {
		return fmt.Errorf("scanSessions: %w", err)
	}

	conversations := groupConversations(sessions)
	for i := range conversations {
		conversations[i].ref = conversationRef{
			provider: provider,
			id:       buildConversationStoreKey(rawDir, provider, conversations[i]),
		}
	}

	if len(changedRawPaths) > 0 {
		transcripts, corpus, err := tryIncrementalRebuild(
			ctx, archiveDir, provider, conversations, changedRawPaths,
		)
		if err == nil {
			return writeCanonicalStoreAtomically(archiveDir, provider, conversations, transcripts, corpus)
		}
		zerolog.Ctx(ctx).Debug().Err(err).Msgf("incremental rebuild failed, falling back to full rebuild")
	}

	transcripts, corpus, err := fullRebuild(ctx, conversations)
	if err != nil {
		return fmt.Errorf("fullRebuild: %w", err)
	}

	if err := writeCanonicalStoreAtomically(
		archiveDir,
		provider,
		conversations,
		transcripts,
		corpus,
	); err != nil {
		return fmt.Errorf("writeCanonicalStoreAtomically: %w", err)
	}

	return nil
}

func fullRebuild(
	ctx context.Context,
	conversations []conversation,
) (map[string]sessionFull, searchCorpus, error) {
	transcripts := make(map[string]sessionFull, len(conversations))
	corpus := searchCorpus{units: make([]searchUnit, 0)}
	for _, conv := range conversations {
		session, err := parseConversationWithSubagents(ctx, conv)
		if err != nil {
			return nil, searchCorpus{}, fmt.Errorf("parseConversationWithSubagents: %w", err)
		}
		key := conv.cacheKey()
		transcripts[key] = session
		corpus.units = append(corpus.units, buildSearchUnits(key, session)...)
	}
	return transcripts, corpus, nil
}

func tryIncrementalRebuild(
	ctx context.Context,
	archiveDir string,
	provider conversationProvider,
	conversations []conversation,
	changedRawPaths []string,
) (map[string]sessionFull, searchCorpus, error) {
	log := zerolog.Ctx(ctx)
	storeDir := providerStoreDir(archiveDir, provider)

	oldCatalog, err := readCatalogFile(filepath.Join(storeDir, "catalog.bin"))
	if err != nil {
		return nil, searchCorpus{}, fmt.Errorf("readCatalogFile: %w", err)
	}

	oldCorpus, err := readSearchFile(filepath.Join(storeDir, "search.bin"))
	if err != nil {
		return nil, searchCorpus{}, fmt.Errorf("readSearchFile: %w", err)
	}

	changedSet := make(map[string]struct{}, len(changedRawPaths))
	for _, path := range changedRawPaths {
		changedSet[path] = struct{}{}
	}

	plan := classifyStoreConversations(conversations, oldCatalog, changedSet)
	oldUnits := groupSearchUnitsByConversation(oldCorpus)

	transcripts := make(map[string]sessionFull, len(conversations))
	corpus := searchCorpus{units: make([]searchUnit, 0)}

	for _, conv := range plan.unchanged {
		key := conv.cacheKey()
		session, err := readTranscriptFile(storeTranscriptPath(storeDir, key))
		if err != nil {
			log.Debug().Err(err).Msgf("incremental rebuild: cannot read transcript %s, re-parsing", key)
			session, err = parseConversationWithSubagents(ctx, conv)
			if err != nil {
				return nil, searchCorpus{}, fmt.Errorf("parseConversationWithSubagents_fallback: %w", err)
			}
			transcripts[key] = session
			corpus.units = append(corpus.units, buildSearchUnits(key, session)...)
			continue
		}
		transcripts[key] = session
		corpus.units = append(corpus.units, oldUnits[key]...)
	}

	toParse := make([]conversation, 0, len(plan.changed)+len(plan.added))
	toParse = append(toParse, plan.changed...)
	toParse = append(toParse, plan.added...)

	for _, conv := range toParse {
		session, err := parseConversationWithSubagents(ctx, conv)
		if err != nil {
			return nil, searchCorpus{}, fmt.Errorf("parseConversationWithSubagents: %w", err)
		}
		key := conv.cacheKey()
		transcripts[key] = session
		corpus.units = append(corpus.units, buildSearchUnits(key, session)...)
	}

	return transcripts, corpus, nil
}

func writeCanonicalStoreAtomically(
	archiveDir string,
	provider conversationProvider,
	conversations []conversation,
	transcripts map[string]sessionFull,
	corpus searchCorpus,
) error {
	storeDir := providerStoreDir(archiveDir, provider)
	storeParent := filepath.Dir(storeDir)
	if err := os.MkdirAll(storeParent, 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	tempDir, err := os.MkdirTemp(storeParent, filepath.Base(storeDir)+"-build-*")
	if err != nil {
		return fmt.Errorf("os.MkdirTemp: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil && !errors.Is(err, os.ErrNotExist) {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msgf("failed to remove %s", tempDir)
		}
	}()

	if err := writeCanonicalStoreDir(tempDir, provider, conversations, transcripts, corpus); err != nil {
		return fmt.Errorf("writeCanonicalStoreDir: %w", err)
	}
	if err := swapCanonicalStoreDir(storeDir, tempDir); err != nil {
		return fmt.Errorf("swapCanonicalStoreDir: %w", err)
	}

	return nil
}

func writeCanonicalStoreDir(
	storeDir string,
	provider conversationProvider,
	conversations []conversation,
	transcripts map[string]sessionFull,
	corpus searchCorpus,
) error {
	if err := os.MkdirAll(filepath.Join(storeDir, "transcripts"), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	for _, conv := range conversations {
		session, ok := transcripts[conv.cacheKey()]
		if !ok {
			return fmt.Errorf("writeCanonicalStoreDir: %w", errors.New("missing transcript for conversation"))
		}
		if err := writeTranscriptFile(storeTranscriptPath(storeDir, conv.cacheKey()), session); err != nil {
			return fmt.Errorf("writeTranscriptFile: %w", err)
		}
	}
	if err := writeCatalogFile(filepath.Join(storeDir, "catalog.bin"), conversations); err != nil {
		return fmt.Errorf("writeCatalogFile: %w", err)
	}
	if err := writeSearchFile(filepath.Join(storeDir, "search.bin"), corpus); err != nil {
		return fmt.Errorf("writeSearchFile: %w", err)
	}
	if err := writeManifest(filepath.Join(storeDir, "manifest.json"), provider); err != nil {
		return fmt.Errorf("writeManifest: %w", err)
	}

	return nil
}

func swapCanonicalStoreDir(storeDir, tempDir string) error {
	exists, err := pathExists(storeDir)
	if err != nil {
		return fmt.Errorf("pathExists: %w", err)
	}
	if !exists {
		if err := os.Rename(tempDir, storeDir); err != nil {
			return fmt.Errorf("os.Rename_new: %w", err)
		}
		return nil
	}

	backupDir, err := reserveTempPath(filepath.Dir(storeDir), filepath.Base(storeDir)+"-backup-*")
	if err != nil {
		return fmt.Errorf("reserveTempPath: %w", err)
	}

	if err := os.Rename(storeDir, backupDir); err != nil {
		return fmt.Errorf("os.Rename_backup: %w", err)
	}
	if err := os.Rename(tempDir, storeDir); err != nil {
		if restoreErr := os.Rename(backupDir, storeDir); restoreErr != nil {
			return fmt.Errorf("os.Rename_store_restore: %v (original: %w)", restoreErr, err)
		}
		return fmt.Errorf("os.Rename_store: %w", err)
	}
	if err := os.RemoveAll(backupDir); err != nil {
		zerolog.Ctx(context.Background()).Warn().Err(err).Msgf("failed to remove %s", backupDir)
	}

	return nil
}

func reserveTempPath(dir, pattern string) (string, error) {
	tempDir, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	if err := os.Remove(tempDir); err != nil {
		return "", fmt.Errorf("os.Remove: %w", err)
	}
	return tempDir, nil
}

func pathExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("os.Stat: %w", err)
	}
	return true, nil
}

func buildConversationStoreKey(
	rawDir string,
	provider conversationProvider,
	conv conversation,
) string {
	hash := sha1.New()
	_, _ = io.WriteString(hash, string(provider))
	_, _ = io.WriteString(hash, "\x00")

	if conv.isSubagent() || conv.name == "" {
		for _, path := range conv.filePaths() {
			rel, err := filepath.Rel(rawDir, path)
			if err != nil {
				rel = path
			}
			_, _ = io.WriteString(hash, filepath.ToSlash(rel))
			_, _ = io.WriteString(hash, "\x00")
		}
		return hex.EncodeToString(hash.Sum(nil))
	}

	projectDir := conversationProjectDir(rawDir, conv)
	_, _ = io.WriteString(hash, projectDir)
	_, _ = io.WriteString(hash, "\x00")
	_, _ = io.WriteString(hash, conv.name)
	return hex.EncodeToString(hash.Sum(nil))
}

func conversationProjectDir(rawDir string, conv conversation) string {
	if len(conv.sessions) == 0 {
		return conv.project.displayName
	}
	rel, err := filepath.Rel(rawDir, conv.sessions[0].filePath)
	if err != nil {
		return conv.project.displayName
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 0 {
		return conv.project.displayName
	}
	return parts[0]
}

func buildSearchUnits(conversationID string, session sessionFull) []searchUnit {
	var units []searchUnit
	for _, msg := range session.messages {
		units = appendSearchUnits(units, conversationID, msg.text)
		for _, call := range msg.toolCalls {
			units = appendSearchUnits(units, conversationID, call.summary)
		}
	}
	return units
}

func appendSearchUnits(units []searchUnit, conversationID, text string) []searchUnit {
	if text == "" {
		return units
	}
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		for _, chunk := range chunkSearchText(trimmed, 160, 48) {
			units = append(units, searchUnit{
				conversationID: conversationID,
				text:           chunk,
			})
		}
	}
	return units
}

func chunkSearchText(text string, maxRunes, overlap int) []string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return []string{text}
	}

	if overlap >= maxRunes {
		overlap = maxRunes / 2
	}

	var chunks []string
	step := maxRunes - overlap
	for start := 0; start < len(runes); start += step {
		end := min(start+maxRunes, len(runes))
		chunks = append(chunks, strings.TrimSpace(string(runes[start:end])))
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func storeTranscriptPath(storeDir, conversationID string) string {
	return filepath.Join(storeDir, "transcripts", conversationID+".bin")
}

func writeManifest(path string, provider conversationProvider) error {
	manifest := storeManifest{
		SchemaVersion:       storeSchemaVersion,
		ProjectionVersion:   storeProjectionVersion,
		SearchCorpusVersion: storeSearchCorpusVersion,
		Provider:            provider,
	}
	return writeJSONFile(path, manifest)
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return os.WriteFile(path, raw, 0o644)
}

func readManifest(path string) (storeManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return storeManifest{}, fmt.Errorf("os.ReadFile: %w", err)
	}
	var manifest storeManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return storeManifest{}, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return manifest, nil
}

func storeNeedsRebuild(
	archiveDir string,
	provider conversationProvider,
) (bool, error) {
	storeDir := providerStoreDir(archiveDir, provider)
	manifest, err := readManifest(filepath.Join(storeDir, "manifest.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return true, fmt.Errorf("readManifest: %w", err)
	}
	if manifest.SchemaVersion != storeSchemaVersion ||
		manifest.ProjectionVersion != storeProjectionVersion ||
		manifest.SearchCorpusVersion != storeSearchCorpusVersion ||
		manifest.Provider != provider {
		return true, nil
	}
	for _, path := range []string{
		filepath.Join(storeDir, "catalog.bin"),
		filepath.Join(storeDir, "search.bin"),
	} {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return true, nil
			}
			return true, fmt.Errorf("os.Stat: %w", err)
		}
	}
	return false, nil
}

func writeCatalogFile(path string, conversations []conversation) error {
	return writeBinaryFile(path, func(w *bufio.Writer) error {
		if _, err := w.WriteString(catalogMagic); err != nil {
			return fmt.Errorf("WriteString: %w", err)
		}
		if err := writeUint(w, uint64(len(conversations))); err != nil {
			return fmt.Errorf("writeUint: %w", err)
		}
		for _, conv := range conversations {
			if err := writeConversation(w, conv); err != nil {
				return fmt.Errorf("writeConversation: %w", err)
			}
		}
		return nil
	})
}

func readCatalogFile(path string) ([]conversation, error) {
	return readBinaryFile(path, func(r *bufio.Reader) ([]conversation, error) {
		magic, err := readFixedString(r, len(catalogMagic))
		if err != nil {
			return nil, fmt.Errorf("readFixedString: %w", err)
		}
		if magic != catalogMagic {
			return nil, fmt.Errorf("readCatalogFile: %w", errInvalidMagic("catalog"))
		}
		count, err := readUint(r)
		if err != nil {
			return nil, fmt.Errorf("readUint: %w", err)
		}
		conversations := make([]conversation, 0, count)
		for range count {
			conv, err := readConversation(r)
			if err != nil {
				return nil, fmt.Errorf("readConversation: %w", err)
			}
			conversations = append(conversations, conv)
		}
		return conversations, nil
	})
}

func writeSearchFile(path string, corpus searchCorpus) error {
	return writeBinaryFile(path, func(w *bufio.Writer) error {
		if _, err := w.WriteString(searchMagic); err != nil {
			return fmt.Errorf("WriteString: %w", err)
		}
		if err := writeUint(w, uint64(len(corpus.units))); err != nil {
			return fmt.Errorf("writeUint: %w", err)
		}
		for _, unit := range corpus.units {
			if err := writeString(w, unit.conversationID); err != nil {
				return fmt.Errorf("writeString_conversationID: %w", err)
			}
			if err := writeString(w, unit.text); err != nil {
				return fmt.Errorf("writeString_text: %w", err)
			}
		}
		return nil
	})
}

func readSearchFile(path string) (searchCorpus, error) {
	return readBinaryFile(path, func(r *bufio.Reader) (searchCorpus, error) {
		magic, err := readFixedString(r, len(searchMagic))
		if err != nil {
			return searchCorpus{}, fmt.Errorf("readFixedString: %w", err)
		}
		if magic != searchMagic {
			return searchCorpus{}, fmt.Errorf("readSearchFile: %w", errInvalidMagic("search"))
		}
		count, err := readUint(r)
		if err != nil {
			return searchCorpus{}, fmt.Errorf("readUint: %w", err)
		}
		corpus := searchCorpus{units: make([]searchUnit, 0, count)}
		for range count {
			conversationID, err := readString(r)
			if err != nil {
				return searchCorpus{}, fmt.Errorf("readString_conversationID: %w", err)
			}
			text, err := readString(r)
			if err != nil {
				return searchCorpus{}, fmt.Errorf("readString_text: %w", err)
			}
			corpus.units = append(corpus.units, searchUnit{
				conversationID: conversationID,
				text:           text,
			})
		}
		return corpus, nil
	})
}

func writeTranscriptFile(path string, session sessionFull) error {
	return writeBinaryFile(path, func(w *bufio.Writer) error {
		if _, err := w.WriteString(transcriptMagic); err != nil {
			return fmt.Errorf("WriteString: %w", err)
		}
		if err := writeSessionFull(w, session); err != nil {
			return fmt.Errorf("writeSessionFull: %w", err)
		}
		return nil
	})
}

func readTranscriptFile(path string) (sessionFull, error) {
	return readBinaryFile(path, func(r *bufio.Reader) (sessionFull, error) {
		magic, err := readFixedString(r, len(transcriptMagic))
		if err != nil {
			return sessionFull{}, fmt.Errorf("readFixedString: %w", err)
		}
		if magic != transcriptMagic {
			return sessionFull{}, fmt.Errorf("readTranscriptFile: %w", errInvalidMagic("transcript"))
		}
		session, err := readSessionFull(r)
		if err != nil {
			return sessionFull{}, fmt.Errorf("readSessionFull: %w", err)
		}
		return session, nil
	})
}

func writeBinaryFile(path string, writeFn func(*bufio.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			zerolog.Ctx(context.Background()).Warn().Err(closeErr).Msgf("failed to close %s", path)
		}
	}()

	writer := bufio.NewWriter(file)
	if err := writeFn(writer); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("writer.Flush: %w", err)
	}
	return nil
}

func readBinaryFile[T any](path string, readFn func(*bufio.Reader) (T, error)) (T, error) {
	file, err := os.Open(path)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = file.Close() }()
	return readFn(bufio.NewReader(file))
}

func writeConversation(w *bufio.Writer, conv conversation) error {
	if err := writeString(w, string(conv.ref.provider)); err != nil {
		return fmt.Errorf("writeString_provider: %w", err)
	}
	if err := writeString(w, conv.ref.id); err != nil {
		return fmt.Errorf("writeString_id: %w", err)
	}
	if err := writeString(w, conv.name); err != nil {
		return fmt.Errorf("writeString_name: %w", err)
	}
	if err := writeString(w, conv.project.displayName); err != nil {
		return fmt.Errorf("writeString_project: %w", err)
	}
	if err := writeUint(w, uint64(len(conv.sessions))); err != nil {
		return fmt.Errorf("writeUint: %w", err)
	}
	for _, session := range conv.sessions {
		if err := writeSessionMeta(w, session); err != nil {
			return fmt.Errorf("writeSessionMeta: %w", err)
		}
	}
	return nil
}

func readConversation(r *bufio.Reader) (conversation, error) {
	providerValue, err := readString(r)
	if err != nil {
		return conversation{}, fmt.Errorf("readString_provider: %w", err)
	}
	id, err := readString(r)
	if err != nil {
		return conversation{}, fmt.Errorf("readString_id: %w", err)
	}
	name, err := readString(r)
	if err != nil {
		return conversation{}, fmt.Errorf("readString_name: %w", err)
	}
	projectName, err := readString(r)
	if err != nil {
		return conversation{}, fmt.Errorf("readString_project: %w", err)
	}
	sessionCount, err := readUint(r)
	if err != nil {
		return conversation{}, fmt.Errorf("readUint: %w", err)
	}
	sessions := make([]sessionMeta, 0, sessionCount)
	for range sessionCount {
		session, err := readSessionMeta(r)
		if err != nil {
			return conversation{}, fmt.Errorf("readSessionMeta: %w", err)
		}
		sessions = append(sessions, session)
	}
	return conversation{
		ref: conversationRef{
			provider: conversationProvider(providerValue),
			id:       id,
		},
		name:     name,
		project:  project{displayName: projectName},
		sessions: sessions,
	}, nil
}

func writeSessionFull(w *bufio.Writer, session sessionFull) error {
	if err := writeSessionMeta(w, session.meta); err != nil {
		return fmt.Errorf("writeSessionMeta: %w", err)
	}
	if err := writeUint(w, uint64(len(session.messages))); err != nil {
		return fmt.Errorf("writeUint: %w", err)
	}
	for _, msg := range session.messages {
		if err := writeMessage(w, msg); err != nil {
			return fmt.Errorf("writeMessage: %w", err)
		}
	}
	return nil
}

func readSessionFull(r *bufio.Reader) (sessionFull, error) {
	meta, err := readSessionMeta(r)
	if err != nil {
		return sessionFull{}, fmt.Errorf("readSessionMeta: %w", err)
	}
	messageCount, err := readUint(r)
	if err != nil {
		return sessionFull{}, fmt.Errorf("readUint: %w", err)
	}
	messages := make([]message, 0, messageCount)
	for range messageCount {
		msg, err := readMessage(r)
		if err != nil {
			return sessionFull{}, fmt.Errorf("readMessage: %w", err)
		}
		messages = append(messages, msg)
	}
	return sessionFull{meta: meta, messages: messages}, nil
}

func writeSessionMeta(w *bufio.Writer, meta sessionMeta) error {
	if err := writeString(w, meta.id); err != nil {
		return fmt.Errorf("writeString_id: %w", err)
	}
	if err := writeString(w, meta.project.displayName); err != nil {
		return fmt.Errorf("writeString_project: %w", err)
	}
	if err := writeString(w, meta.slug); err != nil {
		return fmt.Errorf("writeString_slug: %w", err)
	}
	if err := writeInt(w, meta.timestamp.UnixNano()); err != nil {
		return fmt.Errorf("writeInt_timestamp: %w", err)
	}
	if err := writeInt(w, meta.lastTimestamp.UnixNano()); err != nil {
		return fmt.Errorf("writeInt_lastTimestamp: %w", err)
	}
	if err := writeString(w, meta.cwd); err != nil {
		return fmt.Errorf("writeString_cwd: %w", err)
	}
	if err := writeString(w, meta.gitBranch); err != nil {
		return fmt.Errorf("writeString_gitBranch: %w", err)
	}
	if err := writeString(w, meta.version); err != nil {
		return fmt.Errorf("writeString_version: %w", err)
	}
	if err := writeString(w, meta.model); err != nil {
		return fmt.Errorf("writeString_model: %w", err)
	}
	if err := writeString(w, meta.firstMessage); err != nil {
		return fmt.Errorf("writeString_firstMessage: %w", err)
	}
	if err := writeUint(w, uint64(meta.messageCount)); err != nil {
		return fmt.Errorf("writeUint_messageCount: %w", err)
	}
	if err := writeUint(w, uint64(meta.mainMessageCount)); err != nil {
		return fmt.Errorf("writeUint_mainMessageCount: %w", err)
	}
	if err := writeString(w, meta.filePath); err != nil {
		return fmt.Errorf("writeString_filePath: %w", err)
	}
	if err := writeTokenUsage(w, meta.totalUsage); err != nil {
		return fmt.Errorf("writeTokenUsage: %w", err)
	}
	if err := writeStringIntMap(w, meta.toolCounts); err != nil {
		return fmt.Errorf("writeStringIntMap: %w", err)
	}
	if err := writeBool(w, meta.isSubagent); err != nil {
		return fmt.Errorf("writeBool_isSubagent: %w", err)
	}
	return nil
}

func readSessionMeta(r *bufio.Reader) (sessionMeta, error) {
	id, err := readString(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readString_id: %w", err)
	}
	projectName, err := readString(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readString_project: %w", err)
	}
	slug, err := readString(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readString_slug: %w", err)
	}
	timestampValue, err := readInt(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readInt_timestamp: %w", err)
	}
	lastTimestampValue, err := readInt(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readInt_lastTimestamp: %w", err)
	}
	cwd, err := readString(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readString_cwd: %w", err)
	}
	gitBranch, err := readString(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readString_gitBranch: %w", err)
	}
	version, err := readString(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readString_version: %w", err)
	}
	model, err := readString(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readString_model: %w", err)
	}
	firstMessage, err := readString(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readString_firstMessage: %w", err)
	}
	messageCount, err := readUint(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readUint_messageCount: %w", err)
	}
	mainMessageCount, err := readUint(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readUint_mainMessageCount: %w", err)
	}
	filePath, err := readString(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readString_filePath: %w", err)
	}
	usage, err := readTokenUsage(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readTokenUsage: %w", err)
	}
	toolCounts, err := readStringIntMap(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readStringIntMap: %w", err)
	}
	isSubagent, err := readBool(r)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("readBool_isSubagent: %w", err)
	}

	meta := sessionMeta{
		id:               id,
		project:          project{displayName: projectName},
		slug:             slug,
		cwd:              cwd,
		gitBranch:        gitBranch,
		version:          version,
		model:            model,
		firstMessage:     firstMessage,
		messageCount:     int(messageCount),
		mainMessageCount: int(mainMessageCount),
		filePath:         filePath,
		totalUsage:       usage,
		toolCounts:       toolCounts,
		isSubagent:       isSubagent,
	}
	if timestampValue != 0 {
		meta.timestamp = unixTime(timestampValue)
	}
	if lastTimestampValue != 0 {
		meta.lastTimestamp = unixTime(lastTimestampValue)
	}
	return meta, nil
}

func writeMessage(w *bufio.Writer, msg message) error {
	if err := writeString(w, string(msg.role)); err != nil {
		return fmt.Errorf("writeString_role: %w", err)
	}
	if err := writeString(w, msg.text); err != nil {
		return fmt.Errorf("writeString_text: %w", err)
	}
	if err := writeString(w, msg.thinking); err != nil {
		return fmt.Errorf("writeString_thinking: %w", err)
	}
	if err := writeUint(w, uint64(len(msg.toolCalls))); err != nil {
		return fmt.Errorf("writeUint_toolCalls: %w", err)
	}
	for _, call := range msg.toolCalls {
		if err := writeString(w, call.name); err != nil {
			return fmt.Errorf("writeString_toolCallName: %w", err)
		}
		if err := writeString(w, call.summary); err != nil {
			return fmt.Errorf("writeString_toolCallSummary: %w", err)
		}
	}
	if err := writeUint(w, uint64(len(msg.toolResults))); err != nil {
		return fmt.Errorf("writeUint_toolResults: %w", err)
	}
	for _, result := range msg.toolResults {
		if err := writeToolResult(w, result); err != nil {
			return fmt.Errorf("writeToolResult: %w", err)
		}
	}
	if err := writeBool(w, msg.isSidechain); err != nil {
		return fmt.Errorf("writeBool_isSidechain: %w", err)
	}
	if err := writeBool(w, msg.isAgentDivider); err != nil {
		return fmt.Errorf("writeBool_isAgentDivider: %w", err)
	}
	return nil
}

func readMessage(r *bufio.Reader) (message, error) {
	roleValue, err := readString(r)
	if err != nil {
		return message{}, fmt.Errorf("readString_role: %w", err)
	}
	text, err := readString(r)
	if err != nil {
		return message{}, fmt.Errorf("readString_text: %w", err)
	}
	thinking, err := readString(r)
	if err != nil {
		return message{}, fmt.Errorf("readString_thinking: %w", err)
	}
	callCount, err := readUint(r)
	if err != nil {
		return message{}, fmt.Errorf("readUint_toolCalls: %w", err)
	}
	toolCalls := make([]toolCall, 0, callCount)
	for range callCount {
		name, err := readString(r)
		if err != nil {
			return message{}, fmt.Errorf("readString_toolCallName: %w", err)
		}
		summary, err := readString(r)
		if err != nil {
			return message{}, fmt.Errorf("readString_toolCallSummary: %w", err)
		}
		toolCalls = append(toolCalls, toolCall{name: name, summary: summary})
	}
	resultCount, err := readUint(r)
	if err != nil {
		return message{}, fmt.Errorf("readUint_toolResults: %w", err)
	}
	toolResults := make([]toolResult, 0, resultCount)
	for range resultCount {
		result, err := readToolResult(r)
		if err != nil {
			return message{}, fmt.Errorf("readToolResult: %w", err)
		}
		toolResults = append(toolResults, result)
	}
	isSidechain, err := readBool(r)
	if err != nil {
		return message{}, fmt.Errorf("readBool_isSidechain: %w", err)
	}
	isAgentDivider, err := readBool(r)
	if err != nil {
		return message{}, fmt.Errorf("readBool_isAgentDivider: %w", err)
	}
	return message{
		role:           role(roleValue),
		text:           text,
		thinking:       thinking,
		toolCalls:      toolCalls,
		toolResults:    toolResults,
		isSidechain:    isSidechain,
		isAgentDivider: isAgentDivider,
	}, nil
}

func writeToolResult(w *bufio.Writer, result toolResult) error {
	if err := writeString(w, result.toolName); err != nil {
		return fmt.Errorf("writeString_toolName: %w", err)
	}
	if err := writeString(w, result.toolSummary); err != nil {
		return fmt.Errorf("writeString_toolSummary: %w", err)
	}
	if err := writeString(w, result.content); err != nil {
		return fmt.Errorf("writeString_content: %w", err)
	}
	if err := writeBool(w, result.isError); err != nil {
		return fmt.Errorf("writeBool_isError: %w", err)
	}
	if err := writeUint(w, uint64(len(result.structuredPatch))); err != nil {
		return fmt.Errorf("writeUint_structuredPatch: %w", err)
	}
	for _, hunk := range result.structuredPatch {
		if err := writeDiffHunk(w, hunk); err != nil {
			return fmt.Errorf("writeDiffHunk: %w", err)
		}
	}
	return nil
}

func readToolResult(r *bufio.Reader) (toolResult, error) {
	toolName, err := readString(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readString_toolName: %w", err)
	}
	toolSummary, err := readString(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readString_toolSummary: %w", err)
	}
	content, err := readString(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readString_content: %w", err)
	}
	isError, err := readBool(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readBool_isError: %w", err)
	}
	hunkCount, err := readUint(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readUint_structuredPatch: %w", err)
	}
	patch := make([]diffHunk, 0, hunkCount)
	for range hunkCount {
		hunk, err := readDiffHunk(r)
		if err != nil {
			return toolResult{}, fmt.Errorf("readDiffHunk: %w", err)
		}
		patch = append(patch, hunk)
	}
	return toolResult{
		toolName:        toolName,
		toolSummary:     toolSummary,
		content:         content,
		isError:         isError,
		structuredPatch: patch,
	}, nil
}

func writeDiffHunk(w *bufio.Writer, hunk diffHunk) error {
	for _, value := range []int{hunk.oldStart, hunk.oldLines, hunk.newStart, hunk.newLines} {
		if err := writeInt(w, int64(value)); err != nil {
			return fmt.Errorf("writeInt: %w", err)
		}
	}
	if err := writeUint(w, uint64(len(hunk.lines))); err != nil {
		return fmt.Errorf("writeUint_lines: %w", err)
	}
	for _, line := range hunk.lines {
		if err := writeString(w, line); err != nil {
			return fmt.Errorf("writeString_line: %w", err)
		}
	}
	return nil
}

func readDiffHunk(r *bufio.Reader) (diffHunk, error) {
	oldStart, err := readInt(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readInt_oldStart: %w", err)
	}
	oldLines, err := readInt(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readInt_oldLines: %w", err)
	}
	newStart, err := readInt(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readInt_newStart: %w", err)
	}
	newLines, err := readInt(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readInt_newLines: %w", err)
	}
	lineCount, err := readUint(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readUint_lines: %w", err)
	}
	lines := make([]string, 0, lineCount)
	for range lineCount {
		line, err := readString(r)
		if err != nil {
			return diffHunk{}, fmt.Errorf("readString_line: %w", err)
		}
		lines = append(lines, line)
	}
	return diffHunk{
		oldStart: int(oldStart),
		oldLines: int(oldLines),
		newStart: int(newStart),
		newLines: int(newLines),
		lines:    lines,
	}, nil
}

func writeTokenUsage(w *bufio.Writer, usage tokenUsage) error {
	for _, value := range []int{
		usage.inputTokens,
		usage.cacheCreationInputTokens,
		usage.cacheReadInputTokens,
		usage.outputTokens,
	} {
		if err := writeUint(w, uint64(value)); err != nil {
			return fmt.Errorf("writeUint: %w", err)
		}
	}
	return nil
}

func readTokenUsage(r *bufio.Reader) (tokenUsage, error) {
	values := make([]uint64, 4)
	for i := range values {
		value, err := readUint(r)
		if err != nil {
			return tokenUsage{}, fmt.Errorf("readUint: %w", err)
		}
		values[i] = value
	}
	return tokenUsage{
		inputTokens:              int(values[0]),
		cacheCreationInputTokens: int(values[1]),
		cacheReadInputTokens:     int(values[2]),
		outputTokens:             int(values[3]),
	}, nil
}

func writeStringIntMap(w *bufio.Writer, values map[string]int) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if err := writeUint(w, uint64(len(keys))); err != nil {
		return fmt.Errorf("writeUint: %w", err)
	}
	for _, key := range keys {
		if err := writeString(w, key); err != nil {
			return fmt.Errorf("writeString_key: %w", err)
		}
		if err := writeUint(w, uint64(values[key])); err != nil {
			return fmt.Errorf("writeUint_value: %w", err)
		}
	}
	return nil
}

func readStringIntMap(r *bufio.Reader) (map[string]int, error) {
	count, err := readUint(r)
	if err != nil {
		return nil, fmt.Errorf("readUint: %w", err)
	}
	if count == 0 {
		return nil, nil
	}
	values := make(map[string]int, count)
	for range count {
		key, err := readString(r)
		if err != nil {
			return nil, fmt.Errorf("readString_key: %w", err)
		}
		value, err := readUint(r)
		if err != nil {
			return nil, fmt.Errorf("readUint_value: %w", err)
		}
		values[key] = int(value)
	}
	return values, nil
}

func writeString(w *bufio.Writer, value string) error {
	if err := writeUint(w, uint64(len(value))); err != nil {
		return fmt.Errorf("writeUint: %w", err)
	}
	if _, err := w.WriteString(value); err != nil {
		return fmt.Errorf("WriteString: %w", err)
	}
	return nil
}

func readString(r *bufio.Reader) (string, error) {
	length, err := readUint(r)
	if err != nil {
		return "", fmt.Errorf("readUint: %w", err)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("io.ReadFull: %w", err)
	}
	return string(buf), nil
}

func readFixedString(r *bufio.Reader, length int) (string, error) {
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("io.ReadFull: %w", err)
	}
	return string(buf), nil
}

func writeUint(w *bufio.Writer, value uint64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], value)
	if _, err := w.Write(buf[:n]); err != nil {
		return fmt.Errorf("w.Write: %w", err)
	}
	return nil
}

func readUint(r *bufio.Reader) (uint64, error) {
	value, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, fmt.Errorf("binary.ReadUvarint: %w", err)
	}
	return value, nil
}

func writeInt(w *bufio.Writer, value int64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutVarint(buf[:], value)
	if _, err := w.Write(buf[:n]); err != nil {
		return fmt.Errorf("w.Write: %w", err)
	}
	return nil
}

func readInt(r *bufio.Reader) (int64, error) {
	value, err := binary.ReadVarint(r)
	if err != nil {
		return 0, fmt.Errorf("binary.ReadVarint: %w", err)
	}
	return value, nil
}

func writeBool(w *bufio.Writer, value bool) error {
	byteValue := byte(0)
	if value {
		byteValue = 1
	}
	if err := w.WriteByte(byteValue); err != nil {
		return fmt.Errorf("WriteByte: %w", err)
	}
	return nil
}

func readBool(r *bufio.Reader) (bool, error) {
	value, err := r.ReadByte()
	if err != nil {
		return false, fmt.Errorf("ReadByte: %w", err)
	}
	return value == 1, nil
}

func unixTime(value int64) time.Time {
	return time.Unix(0, value).UTC()
}

func errInvalidMagic(name string) error {
	return fmt.Errorf("invalid %s magic", name)
}
