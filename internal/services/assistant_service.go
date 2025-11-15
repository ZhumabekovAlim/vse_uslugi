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

// answerFromKBViaLLM использует LLM ТОЛЬКО как безопасный форматер ответа из KB.
// Все факты должны браться из bestEntry.Answer, LLM не имеет права придумывать новый функционал.
func (s *AssistantService) answerFromKBViaLLM(
	ctx context.Context,
	params AskParams,
	question string,
	bestEntry ai.KBEntry,
	bestScore int,
) (AskResult, error) {
	llmCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	systemPrompt := "" +
		"Ты — ассистент приложения «Все Услуги».\n" +
		"Тебе дан исходный ответ из базы знаний (это ИСТИНА), а также вопрос пользователя.\n\n" +
		"ТВОЯ ЗАДАЧА:\n" +
		"1) Проверить, что итоговый ответ НЕ противоречит исходному тексту.\n" +
		"2) НЕ ДОБАВЛЯТЬ новый функционал, экраны или возможности, которых НЕТ в исходном ответе.\n" +
		"3) Переписать ответ в следующем формате и стиле:\n" +
		"   - 1) Краткое резюме (1–2 предложения).\n" +
		"   - 2) Пошагово (нумерованный список: 1, 2, 3...).\n" +
		"   - 3) Где найти в приложении (путь по экранам).\n" +
		"   - 4) Советы/важно (1–3 пункта).\n\n" +
		"ОЧЕНЬ ВАЖНО:\n" +
		"- Используй только ту информацию, которая явно содержится в исходном ответе.\n" +
		"- Если вопрос пользователя шире, чем исходный ответ, всё равно не придумывай новых деталей; просто переформулируй и акцентируй то, что есть.\n" +
		"- Никаких ссылок на внешний мир, только логика приложения и текст базы знаний.\n"

	originalAnswer := bestEntry.Answer

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{
			Role: "system",
			Content: fmt.Sprintf(
				"Исходный ответ из базы знаний (источник правды):\n\n%s",
				originalAnswer,
			),
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Вопрос пользователя: %s\nСформируй итоговый ответ по правилам.", question),
		},
	}

	resp, err := s.client.Complete(llmCtx, ChatCompletionRequest{
		Model:       assistantDefaultModel,
		Temperature: 0.1, // ещё более консервативно
		Messages:    messages,
	})

	finalAnswer := strings.TrimSpace(resp.Content)
	if err != nil || finalAnswer == "" {
		// Если LLM всё равно упал — просто отдаем чистый KB-ответ
		return AskResult{
			Answer: originalAnswer,
			Source: "kb",
			KBRefs: []KBRef{{ID: bestEntry.ID, Score: bestScore}},
		}, nil
	}

	return AskResult{
		Answer: finalAnswer,
		Source: "kb+llm", // факты из KB, текст от LLM
		KBRefs: []KBRef{{ID: bestEntry.ID, Score: bestScore}},
	}, nil
}

func (s *AssistantService) Ask(ctx context.Context, params AskParams) (AskResult, error) {
	question := strings.TrimSpace(params.Question)
	if question == "" {
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	var (
		bestEntry  ai.KBEntry
		bestScore  int
		found      bool
		kbHasMatch bool
		snippets   []ai.ScoredEntry
	)

	// 1) Работа с KB, если она есть
	if s.kb != nil {
		bestEntry, bestScore, found = s.kb.FindBestMatch(question)
		kbHasMatch = found && bestScore > 0
		snippets = s.kb.TopEntries(question, params.MaxKB)
	}

	// 2) Если есть хороший матч в KB
	if kbHasMatch {
		// 2.1) LLM есть и включён → используем LLM как безопасный форматер KB
		if params.UseLLM && s.client != nil {
			return s.answerFromKBViaLLM(ctx, params, question, bestEntry, bestScore)
		}

		// 2.2) LLM нет → отдаём KB «как есть»
		return AskResult{
			Answer: bestEntry.Answer,
			Source: "kb",
			KBRefs: []KBRef{{ID: bestEntry.ID, Score: bestScore}},
		}, nil
	}

	// 3) KB не нашла ничего уверенного. Если LLM отключен/нет клиента — остаётся fallback.
	if !params.UseLLM || s.client == nil {
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	// 4) Используем LLM без явного попадания в KB, но всё равно с жёстким системным prompt’ом
	llmCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	systemPrompt := buildSystemPrompt(params.Role, params.Locale)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	// Если есть какие-то сниппеты (даже слабые) – добавим, но они уже >0 score
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

	resp, err := s.client.Complete(llmCtx, ChatCompletionRequest{
		Model:       assistantDefaultModel,
		Temperature: 0.2,
		Messages:    messages,
	})

	answer := strings.TrimSpace(resp.Content)
	if err != nil || answer == "" {
		// KB ничего уверенного не нашла, LLM упал → только честный fallback
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	refs := make([]KBRef, 0, len(snippets))
	for _, snippet := range snippets {
		refs = append(refs, KBRef{ID: snippet.Entry.ID, Score: snippet.Score})
	}

	source := "llm"
	if len(snippets) > 0 {
		source = "kb+llm"
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
