package xart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL   string
	userAgent string
	http      *http.Client
}

type Request struct {
	Method      string
	Path        string
	Query       url.Values
	Headers     map[string]string
	Body        []byte
	ContentType string
	Token       string
	UseV2       bool
}

type APIError struct {
	Status  int
	Code    int
	Message string
	Payload any
}

func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("api error: status=%d code=%d message=%s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("api error: status=%d message=%s", e.Status, e.Message)
}

func NewClient(baseURL, userAgent string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		userAgent: userAgent,
		http: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (c *Client) Do(ctx context.Context, req Request) (any, error) {
	u, err := c.buildURL(req.Path, req.Query, req.Token)
	if err != nil {
		return nil, err
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)
	if req.UseV2 {
		httpReq.Header.Set("API-Version", "v2")
	}
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}
	if len(req.Body) > 0 && httpReq.Header.Get("Content-Type") == "" {
		contentType := req.ContentType
		if contentType == "" {
			contentType = "application/json; charset=UTF-8"
		}
		httpReq.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	payload, parseErr := parsePayload(raw)
	if parseErr != nil {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = parseErr.Error()
		}
		if resp.StatusCode >= 400 {
			return nil, &APIError{
				Status:  resp.StatusCode,
				Message: msg,
			}
		}
		return msg, nil
	}

	if resp.StatusCode >= 400 {
		return nil, extractHTTPError(resp.StatusCode, payload)
	}

	if code, ok := extractCode(payload); ok && code != 0 {
		return nil, &APIError{
			Status:  resp.StatusCode,
			Code:    code,
			Message: extractMessage(payload),
			Payload: payload,
		}
	}

	return payload, nil
}

func (c *Client) JSON(ctx context.Context, method, path string, query url.Values, body any, token string, useV2 bool) (any, error) {
	var rawBody []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode body: %w", err)
		}
		rawBody = data
	}

	return c.Do(ctx, Request{
		Method:  method,
		Path:    path,
		Query:   query,
		Body:    rawBody,
		Token:   token,
		UseV2:   useV2,
		Headers: map[string]string{"Content-Type": "application/json; charset=UTF-8"},
	})
}

func (c *Client) buildURL(path string, extraQuery url.Values, token string) (string, error) {
	var u *url.URL
	var err error

	trimmed := strings.TrimSpace(path)
	switch {
	case strings.HasPrefix(trimmed, "http://"), strings.HasPrefix(trimmed, "https://"):
		u, err = url.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf("invalid url %q: %w", path, err)
		}
	default:
		full := c.baseURL + "/" + strings.TrimLeft(trimmed, "/")
		u, err = url.Parse(full)
		if err != nil {
			return "", fmt.Errorf("invalid path %q: %w", path, err)
		}
	}

	q := u.Query()
	for key, values := range extraQuery {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	if token != "" && q.Get("token") == "" {
		q.Set("token", token)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func parsePayload(raw []byte) (any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{"code": 0}, nil
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func extractCode(payload any) (int, bool) {
	object, ok := payload.(map[string]any)
	if !ok {
		return 0, false
	}
	codeValue, exists := object["code"]
	if !exists {
		return 0, false
	}
	code, ok := toInt(codeValue)
	return code, ok
}

func extractMessage(payload any) string {
	object, ok := payload.(map[string]any)
	if !ok {
		return "unknown api error"
	}
	for _, key := range []string{"message", "detail", "error"} {
		if value, exists := object[key]; exists {
			if msg := strings.TrimSpace(fmt.Sprint(value)); msg != "" {
				return msg
			}
		}
	}
	return "unknown api error"
}

func extractHTTPError(status int, payload any) error {
	if object, ok := payload.(map[string]any); ok {
		code, _ := extractCode(payload)
		return &APIError{
			Status:  status,
			Code:    code,
			Message: extractMessage(object),
			Payload: payload,
		}
	}
	return &APIError{
		Status:  status,
		Message: fmt.Sprintf("http status %d", status),
		Payload: payload,
	}
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		if v == "" {
			return 0, false
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
