package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

type OpenAIClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

func NewOpenAIClient(httpClient *http.Client, apiKey string) *OpenAIClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &OpenAIClient{
		httpClient: httpClient,
		apiKey:     apiKey,
		baseURL:    defaultOpenAIBaseURL,
	}
}

func (c *OpenAIClient) Complete(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	if c == nil {
		return ChatCompletionResponse{}, errors.New("openai client is not configured")
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return ChatCompletionResponse{}, errors.New("openai api key is empty")
	}

	payload := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimRight(c.baseURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return ChatCompletionResponse{}, fmt.Errorf("openai error: status %d: %s", resp.StatusCode, string(data))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("decode response: %w", err)
	}

	if len(parsed.Choices) == 0 {
		return ChatCompletionResponse{}, errors.New("openai returned no choices")
	}

	return ChatCompletionResponse{Content: parsed.Choices[0].Message.Content}, nil
}
