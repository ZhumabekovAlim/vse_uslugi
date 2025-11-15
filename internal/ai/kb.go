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

// FindBestMatch — базовый поиск лучшего совпадения по score.
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

// TopEntries — возвращает только релевантные записи (score > 0),
// отсортированные по убыванию score.
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

// FindByKeywordSubstring — запасной поиск: ищет первую запись,
// у которой какой-либо keyword содержит substr (в нижнем регистре).
func (kb *KnowledgeBase) FindByKeywordSubstring(substr string) (KBEntry, bool) {
	if kb == nil {
		return KBEntry{}, false
	}

	target := strings.ToLower(strings.TrimSpace(substr))
	if target == "" {
		return KBEntry{}, false
	}

	for _, entry := range kb.entries {
		for _, kw := range entry.Keywords {
			if strings.Contains(strings.ToLower(kw), target) {
				return entry, true
			}
		}
	}

	return KBEntry{}, false
}

// scoreEntry — считаем "похожесть" вопроса и записи KB.
// Сначала пытаемся найти keyword целиком как подстроку,
// затем — его "корень" (stem) для разных форм слова.
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

		// 1) Сильное совпадение: keyword целиком в вопросе
		if strings.Contains(question, kw) {
			score += 3
			continue
		}

		// 2) Попробуем корень слова (для форм типа отклик/откликнуться/отклики)
		stem := stemRuWord(kw)
		if stem != "" && strings.Contains(question, stem) {
			score += 2
			continue
		}
	}

	return score
}

// Очень простой "стеммер" для русских слов.
// Нужен, чтобы формы типа "отклик / откликнуться / отклики"
// мапились к одному корню, и т.п.
func stemRuWord(s string) string {
	rs := []rune(s)
	if len(rs) <= 4 {
		// слишком короткие — не трогаем
		return ""
	}

	suffixes := []string{
		// возвратные глаголы
		"ться", "тся", "тись",
		// глагольные суффиксы
		"ировать", "овать", "ивать", "ывать",
		"ить", "ать", "ять", "еть",
		// существительные / прилагательные
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

	// fallback: отрежем последний символ, если слово длинное
	if len(rs) > 5 {
		return string(rs[:len(rs)-1])
	}

	return ""
}

// FindByID — прямой поиск записи по ID.
func (kb *KnowledgeBase) FindByID(id string) (KBEntry, bool) {
	if kb == nil {
		return KBEntry{}, false
	}
	target := strings.TrimSpace(id)
	if target == "" {
		return KBEntry{}, false
	}
	for _, e := range kb.entries {
		if e.ID == target {
			return e, true
		}
	}
	return KBEntry{}, false
}
