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
	assistantConfidenceThreshold = 2
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
	if s.kb == nil {
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	// 1) Сначала ищем по KB
	bestEntry, bestScore, found := s.kb.FindBestMatch(params.Question)
	kbConfident := found && bestScore >= assistantConfidenceThreshold

	// 2) Если LLM не просили или клиента нет — отдаем KB (если уверенно) или fallback
	if !params.UseLLM || s.client == nil {
		if kbConfident {
			return AskResult{Answer: bestEntry.Answer, Source: "kb"}, nil
		}
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	// 3) Готовим контекст для LLM
	snippets := s.kb.TopEntries(params.Question, params.MaxKB)
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildSystemPrompt(params.Role, params.Locale /* экран не передаем */)
	messages := []ChatMessage{
		{Role: "system", Content: prompt},
	}
	if len(snippets) > 0 {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: buildContext(snippets),
		})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: strings.TrimSpace(params.Question)})

	resp, err := s.client.Complete(ctx, ChatCompletionRequest{
		Model:       assistantDefaultModel,
		Temperature: 0.2,
		Messages:    messages,
	})

	// 4) Если LLM упал/пусто — возвращаем KB, если уверенно, иначе fallback
	if err != nil || strings.TrimSpace(resp.Content) == "" {
		if kbConfident {
			return AskResult{Answer: bestEntry.Answer, Source: "kb"}, nil
		}
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	// 5) Успешный LLM; проставляем refs
	answer := strings.TrimSpace(resp.Content)
	source := "llm"
	if len(snippets) > 0 {
		source = "kb+llm"
	}

	refs := make([]KBRef, 0, len(snippets))
	for _, snippet := range snippets {
		refs = append(refs, KBRef{ID: snippet.Entry.ID, Score: snippet.Score})
	}

	return AskResult{Answer: answer, Source: source, KBRefs: refs}, nil
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
