package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"naimuBack/internal/ai"
)

const (
	assistantFallbackAnswer      = "Извините, сейчас не могу помочь. Попробуйте переформулировать вопрос или уточните детали."
	assistantConfidenceThreshold = 1
	assistantDefaultModel        = "gpt-4o-mini"
)

type ChatCompletionClient interface {
	Complete(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error)
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Temperature float32       `json:"temperature,omitempty"`
	Messages    []ChatMessage `json:"messages"`
}

type ChatCompletionResponse struct {
	Content string
}

type AssistantService struct {
	kb      *ai.KnowledgeBase
	client  ChatCompletionClient
	timeout time.Duration
}

type AskParams struct {
	Question string
	Locale   string
	Role     string
	UseLLM   bool
	MaxKB    int
}

type KBRef struct {
	ID    string `json:"id"`
	Score int    `json:"score,omitempty"`
}

type AskResult struct {
	Answer string  `json:"answer"`
	Source string  `json:"source"`
	KBRefs []KBRef `json:"kb_refs,omitempty"`
}

func NewAssistantService(kb *ai.KnowledgeBase, client ChatCompletionClient) *AssistantService {
	return &AssistantService{kb: kb, client: client, timeout: 25 * time.Second}
}

func (s *AssistantService) Ask(ctx context.Context, params AskParams) (AskResult, error) {
	// Если вообще нет KB и нет LLM — только fallback
	if s.kb == nil && (s.client == nil || !params.UseLLM) {
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	var (
		bestEntry  ai.KBEntry
		bestScore  int
		found      bool
		kbHasMatch bool
		snippets   []ai.ScoredEntry
	)

	question := strings.TrimSpace(params.Question)
	if question == "" {
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	// 1) Работа с KB, если она есть
	if s.kb != nil {
		bestEntry, bestScore, found = s.kb.FindBestMatch(question)
		// "хотя бы как-то похоже на что-то из приложения"
		kbHasMatch = found && bestScore > 0

		// Сниппеты только с положительным score (см. обновлённый TopEntries)
		snippets = s.kb.TopEntries(question, params.MaxKB)
	}

	// 2) Если LLM выключен или не настроен — отвечаем только KB / fallback
	if !params.UseLLM || s.client == nil {
		if kbHasMatch {
			refs := []KBRef{{ID: bestEntry.ID, Score: bestScore}}
			return AskResult{
				Answer: bestEntry.Answer,
				Source: "kb",
				KBRefs: refs,
			}, nil
		}

		// KB не нашла ничего похожего -> честный fallback
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	// 3) Используем LLM во всех остальных случаях (универсальное поведение)
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildSystemPrompt(params.Role, params.Locale)

	messages := []ChatMessage{
		{Role: "system", Content: prompt},
	}

	// Если есть релевантные сниппеты — добавляем их отдельным системным сообщением
	if len(snippets) > 0 {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: buildContext(snippets),
		})
	}

	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: question,
	})

	resp, err := s.client.Complete(ctx, ChatCompletionRequest{
		Model:       assistantDefaultModel,
		Temperature: 0.2,
		Messages:    messages,
	})

	answer := strings.TrimSpace(resp.Content)

	// 4) Если LLM отвалился/вернул пустоту — пробуем вернуть хотя бы KB,
	// и только если вообще ничего нет — fallback.
	if err != nil || answer == "" {
		if kbHasMatch {
			refs := []KBRef{{ID: bestEntry.ID, Score: bestScore}}
			return AskResult{
				Answer: bestEntry.Answer,
				Source: "kb",
				KBRefs: refs,
			}, nil
		}

		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	// 5) Успешный ответ LLM
	source := "llm"
	if len(snippets) > 0 {
		source = "kb+llm"
	}

	refs := make([]KBRef, 0, len(snippets))
	for _, snippet := range snippets {
		refs = append(refs, KBRef{
			ID:    snippet.Entry.ID,
			Score: snippet.Score,
		})
	}

	return AskResult{
		Answer: answer,
		Source: source,
		KBRefs: refs,
	}, nil
}

// Было: в шаблоне 3 %s, а передавали 2 → %!s(MISSING)
func buildSystemPrompt(role, locale string) string {
	if strings.TrimSpace(locale) == "" {
		locale = "ru"
	}
	template := "" +
		"Ты — ассистент приложения «Все Услуги». Отвечай ТОЛЬКО по функционалу приложения: объявления, роли, карта, чат, такси/доставка/грузоперевозки, верификация, рейтинги.\n\n" +
		"СТИЛЬ ОТВЕТА (строго придерживайся структуры):\n" +
		"1) Краткое резюме (1–2 предложения, что сделать/где это в целом).\n" +
		"2) Пошагово (нумерованный список с короткими, ясными шагами: 1, 2, 3...).\n" +
		"3) Где найти в приложении (укажи путь вида: Главная → «Заказы» → «Создать объявление»; названия кнопок/вкладок бери из контекста, пиши в кавычках).\n" +
		"4) Советы/важно (1–3 коротких пункта: требования, лимиты, что часто путают).\n\n" +
		"ЕСЛИ КОНТЕКСТ НЕПОЛНЫЙ:\n" +
		"- Не пиши «нет данных» в отрыве. Дай безопасный общий алгоритм в рамках приложения.\n" +
		"- Если отсутствуют точные названия элементов — используй нейтральные формулировки типа «кнопка «Создать объявление» на экране «Заказы»».\n" +
		"- Никогда не выдумывай факты, которых нет в контексте. Лучше укажи вероятные шаги и предложи открыть соответствующий раздел.\n\n" +
		"ФОРМАТИРОВАНИЕ:\n" +
		"- Будь конкретным и понятным, избегай общих фраз.\n" +
		"- Не используй лишнюю воду. В шагах избегай длинных предложений.\n" +
		"- Если вопрос о UI — сначала что делает элемент, затем где находится.\n\n" +
		"МЕТАДАННЫЕ: Роль: %s. Язык: %s.\n" +
		"Используй ТОЛЬКО переданный контекст (сниппеты) и/или общую логику приложения; не выдумывай новые сущности."
	return fmt.Sprintf(template, strings.TrimSpace(role), strings.TrimSpace(locale))
}
func buildContext(snippets []ai.ScoredEntry) string {
	var builder strings.Builder
	builder.WriteString("Контекст знаний:\n")
	for _, snippet := range snippets {
		builder.WriteString(fmt.Sprintf("Ответ: %s\n", snippet.Entry.Answer))
		if len(snippet.Entry.Keywords) > 0 {
			builder.WriteString(fmt.Sprintf("Ключевые слова: %s\n", strings.Join(snippet.Entry.Keywords, ", ")))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}
