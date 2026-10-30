// Package telegram публикует документы в Telegram через Bot API.
// Клиент работает только на исходящих запросах: входящие сообщения не читаются,
// цель публикации фиксирована при создании клиента.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const (
	// apiBaseURL — адрес публичного Bot API; собственный сервер не используется
	apiBaseURL = "https://api.telegram.org"
	// sendTimeout — предел на одну отправку, с запасом на файл в несколько МБ
	sendTimeout = 2 * time.Minute
	// maxAttempts — попытки отправки, включая первую
	maxAttempts = 3
	// defaultRetryDelay — пауза при 429, если сервер не прислал retry_after
	defaultRetryDelay = 5 * time.Second
	// maxResponseBytes — ограничение на чтение ответа, ответы Bot API малы
	maxResponseBytes = 1 << 20
)

// apiResponse — разбор ответа Bot API, включая поля ошибок
type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

// Client отправляет документы в одну заранее заданную группу
type Client struct {
	token      string       // токен бота, в ошибки и логи не попадает
	chatID     string       // единственная разрешённая цель публикации
	apiBase    string       // базовый адрес Bot API
	httpClient *http.Client // клиент с таймаутом больше sendTimeout
}

// New создаёт клиент отправки; пустые значения означают, что Telegram отключён
func New(token, chatID string) *Client {
	return &Client{
		token:   token,
		chatID:  chatID,
		apiBase: apiBaseURL,
		httpClient: &http.Client{
			Timeout: sendTimeout + 30*time.Second,
		},
	}
}

// Enabled сообщает, заданы ли токен и цель отправки
func (c *Client) Enabled() bool {
	return c != nil && c.token != "" && c.chatID != ""
}

// Caption формирует стандартную подпись к файлу отчёта
func Caption(t time.Time) string {
	return "Остатки и цены Ozon, " + t.Format("02.01.2006 15:04")
}

// SendDocument публикует файл в группу. При 429 делает паузу retry_after и повторяет
// отправку, но не более maxAttempts попыток. Токен и адрес запроса в текст ошибок не попадают.
func (c *Client) SendDocument(ctx context.Context, caption, filePath string) error {
	if !c.Enabled() {
		return errors.New("telegram: client is disabled")
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		retryAfter, err := c.send(ctx, caption, filePath)
		if err == nil {
			return nil
		}

		if retryAfter <= 0 || attempt == maxAttempts {
			return err
		}

		timer := time.NewTimer(retryAfter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("telegram: send cancelled: %w", ctx.Err())
		case <-timer.C:
		}
	}

	return errors.New("telegram: send failed")
}

// send выполняет одну попытку отправки; возвращает паузу перед повтором при 429
func (c *Client) send(ctx context.Context, caption, filePath string) (time.Duration, error) {
	body, contentType, err := buildMultipartBody(c.chatID, caption, filePath)
	if err != nil {
		return 0, err
	}

	endpoint := c.apiBase + "/bot" + c.token + "/sendDocument"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return 0, fmt.Errorf("telegram: build request: %w", transportError(err))
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("telegram: request failed: %w", transportError(err))
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	decErr := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&apiResp)
	if decErr != nil {
		return 0, fmt.Errorf("telegram: unexpected response status %d", resp.StatusCode)
	}

	if apiResp.OK {
		return 0, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests || apiResp.ErrorCode == http.StatusTooManyRequests {
		delay := time.Duration(apiResp.Parameters.RetryAfter) * time.Second
		if delay <= 0 {
			delay = defaultRetryDelay
		}
		return delay, fmt.Errorf("telegram: rate limited, retry after %s", delay)
	}

	if apiResp.Description != "" {
		return 0, fmt.Errorf("telegram: %s", apiResp.Description)
	}

	return 0, fmt.Errorf("telegram: unexpected response status %d", resp.StatusCode)
}

// buildMultipartBody собирает тело multipart-запроса с файлом отчёта
func buildMultipartBody(chatID, caption, filePath string) (*bytes.Buffer, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("telegram: open report: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("chat_id", chatID); err != nil {
		return nil, "", fmt.Errorf("telegram: build multipart: %w", err)
	}
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return nil, "", fmt.Errorf("telegram: build multipart: %w", err)
		}
	}

	part, err := writer.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return nil, "", fmt.Errorf("telegram: build multipart: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, "", fmt.Errorf("telegram: read report: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("telegram: build multipart: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

// transportError отбрасывает часть ошибки с URL: в адресе запроса находится токен бота
func transportError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err
	}
	return err
}
