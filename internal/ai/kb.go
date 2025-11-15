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
		if score <= 0 {
			// нерелевантные записи не тащим в контекст LLM
			continue
		}
		scored = append(scored, ScoredEntry{
			Entry: entry,
			Score: score,
		})
	}

	if len(scored) == 0 {
		return nil
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
	if lowerQuestion == "" {
		return 0
	}

	question := lowerQuestion
	score := 0

	for _, kw := range entry.Keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw == "" {
			continue
		}

		// 1) Сильное совпадение: ключевое слово целиком как подстрока
		if strings.Contains(question, kw) {
			score += 3
			continue
		}

		// 2) Попробуем корень (stem) для русского слова
		stem := stemRuWord(kw)
		if stem != "" && strings.Contains(question, stem) {
			score += 2
			continue
		}
	}

	return score
}

// Очень простой стеммер для русских слов: обрезает частые суффиксы,
// чтобы "отклик", "откликнуться", "отклики" давали общий корень.
func stemRuWord(s string) string {
	rs := []rune(s)
	if len(rs) <= 4 {
		// слишком короткие слова лучше не трогать
		return ""
	}

	suffixes := []string{
		// возвратные глаголы
		"ться", "тся", "тись",
		// глагольные суффиксы
		"ировать", "ировать", "овать", "ивать", "ывать",
		"ить", "ать", "ять", "еть",
		// типичные существительные / прилагательные
		"ки", "ка", "ку", "кой", "ках",
		"ов", "ев", "ом", "ами", "ями",
		"ый", "ий", "ой",
		"ая", "яя",
		"ое", "ее",
		"ые", "ие",
	}

	for _, suf := range suffixes {
		sr := []rune(suf)
		if len(rs) > len(sr)+2 && string(rs[len(rs)-len(sr):]) == suf {
			return string(rs[:len(rs)-len(sr)])
		}
	}

	// fallback: отрезаем один символ, если слово не станет слишком коротким
	if len(rs) > 5 {
		return string(rs[:len(rs)-1])
	}

	return ""
}
