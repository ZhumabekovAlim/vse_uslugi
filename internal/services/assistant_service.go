package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"naimuBack/internal/ai"
)

const (
	assistantFallbackAnswer = "Извините, сейчас не могу помочь. Попробуйте переформулировать вопрос или уточните детали."
	assistantDefaultModel   = "gpt-4o-mini"
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
	Source string  `json:"source"` // "kb", "llm", "kb+llm", "fallback"
	KBRefs []KBRef `json:"kb_refs,omitempty"`
}

func NewAssistantService(kb *ai.KnowledgeBase, client ChatCompletionClient) *AssistantService {
	return &AssistantService{
		kb:      kb,
		client:  client,
		timeout: 25 * time.Second,
	}
}

// Главная функция ассистента.
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

	lowerQ := strings.ToLower(question)

	// 1) Работа с KB, если она есть
	if s.kb != nil {
		bestEntry, bestScore, found = s.kb.FindBestMatch(question)
		kbHasMatch = found && bestScore > 0
		snippets = s.kb.TopEntries(question, params.MaxKB)

		// ⚡ Если обычный матч не найден — пробуем доменные правила.
		if !kbHasMatch {
			if entry, ok := pickDomainKBEntry(s.kb, lowerQ, params.Role); ok {
				bestEntry = entry
				bestScore = 1
				kbHasMatch = true
				snippets = []ai.ScoredEntry{
					{Entry: entry, Score: 1},
				}
			}
		}
	}

	// 2) Если есть матч в KB
	if kbHasMatch {
		// 2.1) LLM включен — используем его как безопасный форматер KB-ответа
		if params.UseLLM && s.client != nil {
			return s.answerFromKBViaLLM(ctx, params, question, bestEntry, bestScore)
		}

		// 2.2) LLM нет — просто отдаем KB
		return AskResult{
			Answer: bestEntry.Answer,
			Source: "kb",
			KBRefs: []KBRef{{ID: bestEntry.ID, Score: bestScore}},
		}, nil
	}

	// 3) KB не помогла. Если LLM отключен/нет клиента — честный fallback.
	if !params.UseLLM || s.client == nil {
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	// 4) Используем LLM без уверенного попадания в KB, но с системным prompt'ом
	llmCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildSystemPrompt(params.Role, params.Locale)

	messages := []ChatMessage{
		{Role: "system", Content: prompt},
	}

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
		// KB ничего не нашла, LLM отвалился → только fallback
		return AskResult{Answer: assistantFallbackAnswer, Source: "fallback"}, nil
	}

	refs := make([]KBRef, 0, len(snippets))
	for _, snippet := range snippets {
		refs = append(refs, KBRef{
			ID:    snippet.Entry.ID,
			Score: snippet.Score,
		})
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

// pickDomainKBEntry — "умный" доменный fallback.
// Если обычный score не нашел уверенный матч, но в вопросе есть ключевые слова
// (такси, доставка, объявление, подписка, заказ, чат, город, профиль),
// пробуем подобрать подходящий KBEntry по заранее заданным правилам.
// Учитываем роль (passenger/driver/courier/user), чтобы выбирать более точный ответ.
func pickDomainKBEntry(kb *ai.KnowledgeBase, lowerQ, role string) (ai.KBEntry, bool) {
	if kb == nil {
		return ai.KBEntry{}, false
	}

	role = strings.ToLower(strings.TrimSpace(role))

	type domainRule struct {
		substr    string   // что ищем в вопросе (в lowerQ)
		preferIDs []string // какие KB id пробуем в приоритете
	}

	rules := make([]domainRule, 0, 16)

	// --- Такси ---
	if strings.Contains(lowerQ, "такси") {
		switch role {
		case "driver", "водитель":
			rules = append(rules, domainRule{
				substr: "такси",
				preferIDs: []string{
					"taxi_driver_home",
					"taxi_driver_accept_order",
					"taxi_driver_intercity",
					"taxi_driver_history",
					"taxi_driver_profile",
				},
			})
		case "courier", "курьер":
			rules = append(rules, domainRule{
				substr: "такси",
				preferIDs: []string{
					"taxi_courier_home",
					"taxi_courier_history",
					"taxi_courier_profile",
				},
			})
		default: // пассажир / пользователь
			rules = append(rules, domainRule{
				substr: "такси",
				preferIDs: []string{
					"taxi_passenger_order",
					"taxi_passenger_intercity",
					"taxi_passenger_history",
				},
			})
		}
	}

	// --- Доставка / курьеры ---
	if strings.Contains(lowerQ, "доставк") || strings.Contains(lowerQ, "курьер") {
		rules = append(rules,
			domainRule{
				substr: "доставк",
				preferIDs: []string{
					"taxi_sender_order",
					"taxi_sender_history",
				},
			},
			domainRule{
				substr: "курьер",
				preferIDs: []string{
					"taxi_courier_become",
					"taxi_courier_home",
					"taxi_courier_history",
					"taxi_courier_profile",
				},
			},
		)
	}

	// --- Объявления ---
	if strings.Contains(lowerQ, "объявлен") {
		rules = append(rules, domainRule{
			substr: "объявлен",
			preferIDs: []string{
				"ad_create",
				"ad_create_category",
				"ad_edit",
				"ad_view",
				"ad_respond",
				"ad_reviews_view",
				"ad_executor_other_ads",
				"ad_location_view",
			},
		})
	}

	// --- Подписки ---
	if strings.Contains(lowerQ, "подписк") {
		rules = append(rules, domainRule{
			substr: "подписк",
			preferIDs: []string{
				"profile_subscriptions",
				"ad_respond_balance",
			},
		})
	}

	// --- Заказы ---
	if strings.Contains(lowerQ, "заказ") {
		rules = append(rules, domainRule{
			substr: "заказ",
			preferIDs: []string{
				"order_create",
				"order_track",
				"order_cancel",
				"order_pay",
				"taxi_passenger_order",
				"taxi_sender_order",
			},
		})
	}

	// --- Чаты / сообщения ---
	if strings.Contains(lowerQ, "чат") || strings.Contains(lowerQ, "сообщен") {
		rules = append(rules, domainRule{
			substr: "чат",
			preferIDs: []string{
				"chat_list",
				"chat_open",
				"chat_send_message",
				"chat_send_photo",
				"chat_ad_chats",
				"ad_respond",
			},
		})
	}

	// --- Город / города ---
	if strings.Contains(lowerQ, "город") {
		rules = append(rules, domainRule{
			substr: "город",
			preferIDs: []string{
				"city_select",
				"taxi_passenger_intercity",
			},
		})
	}

	// --- Профиль / аккаунт ---
	if strings.Contains(lowerQ, "профил") || strings.Contains(lowerQ, "аккаунт") {
		rules = append(rules, domainRule{
			substr: "профил",
			preferIDs: []string{
				"profile_edit",
				"profile_change_password",
				"profile_change_avatar",
				"profile_my_ads",
				"profile_my_orders",
				"profile_favorites",
				"profile_payment_history",
				"profile_settings",
				"profile_help",
				"profile_privacy_policy",
				"delete_account",
			},
		})
	}

	entries := kb.Entries()

	// 1) Пробуем пройти по всем правилам и найти приоритетный ID
	for _, rule := range rules {
		if !strings.Contains(lowerQ, rule.substr) {
			continue
		}

		// Сначала пытаемся найти по приоритетным ID
		for _, preferredID := range rule.preferIDs {
			for _, e := range entries {
				if e.ID == preferredID {
					return e, true
				}
			}
		}

		// Если ни один preferredID не нашелся (например, его удалили из KB),
		// пробуем общий поиск по подстроке keyword.
		if entry, ok := kb.FindByKeywordSubstring(rule.substr); ok {
			return entry, true
		}
	}

	return ai.KBEntry{}, false
}

// answerFromKBViaLLM — LLM используется ТОЛЬКО как форматер/проверяющий KB-ответа.
// Все факты должны оставаться из bestEntry.Answer.
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
		"3) Переписать ответ в формате:\n" +
		"   1) Краткое резюме (1–2 предложения).\n" +
		"   2) Пошагово (нумерованный список 1, 2, 3...).\n" +
		"   3) Где найти в приложении (путь по экранам).\n" +
		"   4) Советы/важно (1–3 пункта).\n\n" +
		"ОЧЕНЬ ВАЖНО:\n" +
		"- Используй только явную информацию из исходного ответа.\n" +
		"- Если вопрос шире, чем исходный ответ, не придумывай лишние детали.\n" +
		"- Не упоминай внешний мир, только логику приложения и текст базы знаний.\n"

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
			Role: "user",
			Content: fmt.Sprintf(
				"Вопрос пользователя: %s\nСформируй итоговый ответ по правилам.",
				question,
			),
		},
	}

	resp, err := s.client.Complete(llmCtx, ChatCompletionRequest{
		Model:       assistantDefaultModel,
		Temperature: 0.1, // максимально консервативно
		Messages:    messages,
	})

	finalAnswer := strings.TrimSpace(resp.Content)
	if err != nil || finalAnswer == "" {
		// Если LLM упал — просто отдаем KB-ответ
		return AskResult{
			Answer: originalAnswer,
			Source: "kb",
			KBRefs: []KBRef{{ID: bestEntry.ID, Score: bestScore}},
		}, nil
	}

	return AskResult{
		Answer: finalAnswer,
		Source: "kb+llm",
		KBRefs: []KBRef{{ID: bestEntry.ID, Score: bestScore}},
	}, nil
}

// buildSystemPrompt — системный prompt, когда KB не дала уверенного ответа.
func buildSystemPrompt(role, locale string) string {
	if strings.TrimSpace(locale) == "" {
		locale = "ru"
	}
	template := "" +
		"Ты — ассистент приложения «Все Услуги». Отвечай ТОЛЬКО по функционалу приложения: объявления, роли, карта, чат, такси/доставка/грузоперевозки, верификация, рейтинги.\n\n" +
		"СТИЛЬ ОТВЕТА (строго придерживайся структуры):\n" +
		"1) Краткое резюме (1–2 предложения, что сделать/где это в целом).\n" +
		"2) Пошагово (нумерованный список с короткими шагами: 1, 2, 3...).\n" +
		"3) Где найти в приложении (путь вида: Главная → «Заказы» → «Создать объявление»; названия кнопок/вкладок бери из контекста, пиши в кавычках).\n" +
		"4) Советы/важно (1–3 коротких пункта: требования, лимиты, что часто путают).\n\n" +
		"ЕСЛИ КОНТЕКСТ НЕПОЛНЫЙ:\n" +
		"- Не пиши «нет данных» в отрыве. Дай безопасный общий алгоритм в рамках приложения.\n" +
		"- Если отсутствуют точные названия элементов — используй нейтральные формулировки.\n" +
		"- Никогда не выдумывай факты, которых нет в контексте. Лучше опиши вероятные шаги.\n\n" +
		"ФОРМАТИРОВАНИЕ:\n" +
		"- Будь конкретным и понятным, избегай воды.\n" +
		"- В шагах избегай длинных предложений.\n\n" +
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
