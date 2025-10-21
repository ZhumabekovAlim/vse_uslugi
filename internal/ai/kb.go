package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type KBEntry struct {
	ID       string   `json:"id"`
	Keywords []string `json:"keywords"`
	Answer   string   `json:"answer"`
	Deeplink string   `json:"deeplink,omitempty"`
}

type KnowledgeBase struct {
	entries []KBEntry
}

type ScoredEntry struct {
	Entry KBEntry
	Score int
}

func LoadKnowledgeBase(path string) (*KnowledgeBase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read knowledge base: %w", err)
	}

	var entries []KBEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse knowledge base: %w", err)
	}

	return &KnowledgeBase{entries: entries}, nil
}

func (kb *KnowledgeBase) Entries() []KBEntry {
	if kb == nil {
		return nil
	}
	result := make([]KBEntry, len(kb.entries))
	copy(result, kb.entries)
	return result
}

func (kb *KnowledgeBase) FindBestMatch(question string) (KBEntry, int, bool) {
	if kb == nil {
		return KBEntry{}, 0, false
	}

	lowerQuestion := strings.ToLower(question)

	var best KBEntry
	bestScore := 0
	found := false

	for _, entry := range kb.entries {
		score := scoreEntry(entry, lowerQuestion)
		if !found || score > bestScore {
			best = entry
			bestScore = score
			found = true
		}
	}

	return best, bestScore, found
}

func (kb *KnowledgeBase) TopEntries(question string, limit int) []ScoredEntry {
	if kb == nil || limit <= 0 {
		return nil
	}

	lowerQuestion := strings.ToLower(question)

	scored := make([]ScoredEntry, 0, len(kb.entries))
	for _, entry := range kb.entries {
		score := scoreEntry(entry, lowerQuestion)
		scored = append(scored, ScoredEntry{
			Entry: entry,
			Score: score,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].Entry.ID < scored[j].Entry.ID
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	return scored
}

func scoreEntry(entry KBEntry, lowerQuestion string) int {
	score := 0
	if lowerQuestion == "" {
		return score
	}

	question := lowerQuestion
	for _, keyword := range entry.Keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword == "" {
			continue
		}
		if strings.Contains(question, keyword) {
			score++
		}
	}

	return score
}
