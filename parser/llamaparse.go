package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type LlamaParseParser struct {
	cfg LlamaParseConfig
}

func NewLlamaParseParser(cfg LlamaParseConfig) *LlamaParseParser {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.cloud.llamaindex.ai/api/parsing"
	}
	return &LlamaParseParser{cfg: cfg}
}

func (p *LlamaParseParser) SupportedFormats() []string {
	return []string{"doc", "xls", "ppt", "pdf", "docx", "xlsx", "pptx"}
}

func (p *LlamaParseParser) Parse(ctx context.Context, path string) (*ParseResult, error) {
	if p.cfg.APIKey == "" {
		return nil, fmt.Errorf("LlamaParse API key not configured")
	}

	// Upload file
	jobID, err := p.uploadFile(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("uploading to LlamaParse: %w", err)
	}

	// Poll for completion
	result, err := p.pollResult(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("getting LlamaParse result: %w", err)
	}

	// Parse the markdown result into sections
	sections := splitPageIntoSections(result, 1)

	return &ParseResult{
		Sections: sections,
		Method:   "llamaparse",
	}, nil
}

func (p *LlamaParseParser) uploadFile(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/upload", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}

func (p *LlamaParseParser) pollResult(ctx context.Context, jobID string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i < 60; i++ { // max ~5 minutes
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
		}

		req, err := http.NewRequestWithContext(ctx, "GET",
			fmt.Sprintf("%s/job/%s/result/markdown", p.cfg.BaseURL, jobID), nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var result struct {
				Markdown string `json:"markdown"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return string(body), nil // raw text fallback
			}
			return result.Markdown, nil
		}

		if resp.StatusCode != http.StatusAccepted {
			return "", fmt.Errorf("LlamaParse error %d: %s", resp.StatusCode, string(body))
		}
	}

	return "", fmt.Errorf("LlamaParse job timed out")
}
