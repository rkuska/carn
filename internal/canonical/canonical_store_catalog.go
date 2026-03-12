package canonical

import (
	"bufio"
	"fmt"
)

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

func writeConversation(w *bufio.Writer, conv conversation) error {
	if err := writeString(w, string(conv.Ref.Provider)); err != nil {
		return fmt.Errorf("writeString_provider: %w", err)
	}
	if err := writeString(w, conv.Ref.ID); err != nil {
		return fmt.Errorf("writeString_id: %w", err)
	}
	if err := writeString(w, conv.Name); err != nil {
		return fmt.Errorf("writeString_name: %w", err)
	}
	if err := writeString(w, conv.Project.DisplayName); err != nil {
		return fmt.Errorf("writeString_project: %w", err)
	}
	if err := writeUint(w, uint64(len(conv.Sessions))); err != nil {
		return fmt.Errorf("writeUint_sessions: %w", err)
	}
	for _, session := range conv.Sessions {
		if err := writeSessionMeta(w, session); err != nil {
			return fmt.Errorf("writeSessionMeta: %w", err)
		}
	}
	if err := writeUint(w, uint64(conv.PlanCount)); err != nil {
		return fmt.Errorf("writeUint_planCount: %w", err)
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
	planCount, err := readUint(r)
	if err != nil {
		return conversation{}, fmt.Errorf("readUint_planCount: %w", err)
	}
	return conversation{
		Ref: conversationRef{
			Provider: conversationProvider(providerValue),
			ID:       id,
		},
		Name:      name,
		Project:   project{DisplayName: projectName},
		Sessions:  sessions,
		PlanCount: int(planCount),
	}, nil
}
