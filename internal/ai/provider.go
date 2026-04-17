package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dedikodu-kazani/backend/internal/config"
)

type Provider interface {
	GenerateReply(ctx context.Context, persona string, history []string, latest string) (string, error)
}

func New(cfg config.Config) Provider {
	client := &http.Client{Timeout: 25 * time.Second}
	switch strings.ToLower(cfg.AIProvider) {
	case "anthropic":
		return &Anthropic{client: client, apiKey: cfg.AnthropicKey, model: cfg.AnthropicModel}
	case "gemini":
		return &Gemini{client: client, apiKey: cfg.GeminiKey, model: cfg.GeminiModel}
	default:
		return &OpenAI{client: client, apiKey: cfg.OpenAIKey, model: cfg.OpenAIModel}
	}
}

func systemPrompt(persona string, history []string, latest string) string {
	if persona == "" {
		persona = "Gazlayici"
	}
	return fmt.Sprintf(`Sen Dedikodu Kazani uygulamasinda "%s" personasina sahip bir AI sohbet botusun.
Turkce cevap ver. Kisa, komik, iğneleyici ama hakaret ve nefret soylemi uretmeyen cevaplar yaz.
Kimseyi hedef gostermekten, ozel veri istemekten ve suca tesvikten kacin.
Son mesajlar: %s
Kullanicinin son mesaji: %s`, persona, strings.Join(history, " | "), latest)
}

type OpenAI struct {
	client *http.Client
	apiKey string
	model  string
}

func (p *OpenAI) GenerateReply(ctx context.Context, persona string, history []string, latest string) (string, error) {
	if p.apiKey == "" {
		return "", errors.New("OPENAI_API_KEY is empty")
	}
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt(persona, history, latest)},
			{"role": "user", "content": latest},
		},
		"temperature": 0.85,
		"max_tokens":  140,
	}
	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := p.post(ctx, "https://api.openai.com/v1/chat/completions", body, &res); err != nil {
		return "", err
	}
	if len(res.Choices) == 0 {
		return "", errors.New("openai returned no choices")
	}
	return strings.TrimSpace(res.Choices[0].Message.Content), nil
}

func (p *OpenAI) post(ctx context.Context, url string, body any, out any) error {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("openai status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type Anthropic struct {
	client *http.Client
	apiKey string
	model  string
}

func (p *Anthropic) GenerateReply(ctx context.Context, persona string, history []string, latest string) (string, error) {
	if p.apiKey == "" {
		return "", errors.New("ANTHROPIC_API_KEY is empty")
	}
	body := map[string]any{
		"model":      p.model,
		"max_tokens": 140,
		"system":     systemPrompt(persona, history, latest),
		"messages": []map[string]string{
			{"role": "user", "content": latest},
		},
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(data))
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("anthropic status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if len(res.Content) == 0 {
		return "", errors.New("anthropic returned no content")
	}
	return strings.TrimSpace(res.Content[0].Text), nil
}

type Gemini struct {
	client *http.Client
	apiKey string
	model  string
}

func (p *Gemini) GenerateReply(ctx context.Context, persona string, history []string, latest string) (string, error) {
	if p.apiKey == "" {
		return "", errors.New("GEMINI_API_KEY is empty")
	}
	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": systemPrompt(persona, history, latest)}}},
		},
		"generationConfig": map[string]any{"temperature": 0.85, "maxOutputTokens": 140},
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.model, p.apiKey)
	var res struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("gemini returned no content")
	}
	return strings.TrimSpace(res.Candidates[0].Content.Parts[0].Text), nil
}
