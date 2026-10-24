package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	URL   string `json:"url"`
	Model string `json:"model"`
	Token string `json:"token"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "iai", "config.json")
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lendo config %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parseando config: %w", err)
	}
	if c.URL == "" {
		c.URL = "http://127.0.0.1:2211/v1"
	}
	if c.Model == "" {
		c.Model = "opencode/big-pickle"
	}
	return &c, nil
}

type Client struct {
	cfg    *Config
	client *http.Client
}

func NewClient(cfg *Config) *Client {
	return &Client{cfg: cfg, client: &http.Client{Timeout: 300 * time.Second}}
}

func (c *Client) Chat(systemPrompt, userContent string) (string, error) {
	req := ChatRequest{
		Model: c.cfg.Model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Stream: false,
	}
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", strings.TrimRight(c.cfg.URL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("chamando LLM: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("LLM retornou %d", resp.StatusCode)
	}
	var cr ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", fmt.Errorf("decodando resposta: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("resposta vazia do LLM")
	}
	return cr.Choices[0].Message.Content, nil
}

func (c *Client) ChatStream(systemPrompt, userContent string) (string, error) {
	req := ChatRequest{
		Model: c.cfg.Model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Stream: true,
	}
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", strings.TrimRight(c.cfg.URL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var full strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(data), &chunk) == nil && len(chunk.Choices) > 0 {
			full.WriteString(chunk.Choices[0].Delta.Content)
		}
	}
	return full.String(), scanner.Err()
}
