package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"naimuBack/internal/ai"
)

const (
	assistantFallbackAnswer      = "Извините, сейчас не могу помочь. Откройте нужный экран приложения."
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
	Screen   string
	Role     string
	UseLLM   bool
	MaxKB    int
}

type KBRef struct {
	ID       string `json:"id"`
	Screen   string `json:"screen,omitempty"`
	Score    int    `json:"score,omitempty"`
	Deeplink string `json:"deeplink,omitempty"`
}

type AskResult struct {
	Answer   string  `json:"answer"`
	Source   string  `json:"source"`
	KBRefs   []KBRef `json:"kb_refs,omitempty"`
	Deeplink string  `json:"deeplink,omitempty"`
}

func NewAssistantService(kb *ai.KnowledgeBase, client ChatCompletionClient) *AssistantService {
	return &AssistantService{kb: kb, client: client, timeout: 15 * time.Second}
}

func (s *AssistantService) Ask(ctx context.Context, params AskParams) (AskResult, error) {
	if s.kb == nil {
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	bestEntry, bestScore, found := s.kb.FindBestMatch(params.Question, params.Screen)
	if found && bestScore >= assistantConfidenceThreshold {
		refs := []KBRef{{
			ID:       bestEntry.ID,
			Screen:   bestEntry.Screen,
			Score:    bestScore,
			Deeplink: bestEntry.Deeplink,
		}}
		result := AskResult{
			Answer: bestEntry.Answer,
			Source: "kb",
			KBRefs: refs,
		}
		if bestEntry.Deeplink != "" {
			result.Deeplink = bestEntry.Deeplink
		}
		return result, nil
	}

	if !params.UseLLM || s.client == nil {
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	snippets := s.kb.TopEntries(params.Question, params.Screen, params.MaxKB)
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildSystemPrompt(params.Role, params.Locale, params.Screen)
	messages := []ChatMessage{
		{
			Role:    "system",
			Content: prompt,
		},
	}

	if len(snippets) > 0 {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: buildContext(snippets),
		})
	}

	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: strings.TrimSpace(params.Question),
	})

	resp, err := s.client.Complete(ctx, ChatCompletionRequest{
		Model:       assistantDefaultModel,
		Temperature: 0.2,
		Messages:    messages,
	})
	if err != nil || strings.TrimSpace(resp.Content) == "" {
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	answer := strings.TrimSpace(resp.Content)
	source := "llm"
	if len(snippets) > 0 {
		source = "kb+llm"
	}

	refs := make([]KBRef, 0, len(snippets))
	for _, snippet := range snippets {
		refs = append(refs, KBRef{
			ID:       snippet.Entry.ID,
			Screen:   snippet.Entry.Screen,
			Score:    snippet.Score,
			Deeplink: snippet.Entry.Deeplink,
		})
	}

	result := AskResult{
		Answer: answer,
		Source: source,
		KBRefs: refs,
	}
	if len(snippets) > 0 && snippets[0].Entry.Deeplink != "" {
		result.Deeplink = snippets[0].Entry.Deeplink
	}
	return result, nil
}

func buildSystemPrompt(role, locale, screen string) string {
	if strings.TrimSpace(locale) == "" {
		locale = "ru"
	}

	template := "Ты — ассистент приложения «Все Услуги». Отвечай только по функционалу приложения (объявления, роли, карта, чат, такси/доставка/грузоперевозки, верификация, рейтинги).\n" +
		"Пиши кратко и по шагам (1–2–3).\n" +
		"Если вопрос о кнопке/элементе UI — сначала что делает, затем где находится.\n" +
		"Если нет точных данных — честно скажи, что информации нет, предложи открыть нужный экран.\n" +
		"Роль: %s. Язык: %s. Экран: %s.\n" +
		"Используй только переданный контекст (сниппеты) и не выдумывай."

	return fmt.Sprintf(template, strings.TrimSpace(role), strings.TrimSpace(locale), strings.TrimSpace(screen))
}

func buildContext(snippets []ai.ScoredEntry) string {
	var builder strings.Builder
	builder.WriteString("Контекст знаний:\n")
	for idx, snippet := range snippets {
		builder.WriteString(fmt.Sprintf("%d) Экран: %s\n", idx+1, snippet.Entry.Screen))
		builder.WriteString(fmt.Sprintf("Ответ: %s\n", snippet.Entry.Answer))
		if len(snippet.Entry.Keywords) > 0 {
			builder.WriteString(fmt.Sprintf("Ключевые слова: %s\n", strings.Join(snippet.Entry.Keywords, ", ")))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}
