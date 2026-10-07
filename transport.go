package payriff

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxErrorBody = 500

type apiRequest struct {
	method           string
	path             string
	query            [][2]string
	headers          map[string]string
	body             any
	merchantEnvelope bool
}

type transport struct {
	baseURL    string
	appKey     string
	merchantID string
	httpClient *http.Client
}

type envelope struct {
	Code       *string         `json:"code"`
	Message    *string         `json:"message"`
	ResponseID *string         `json:"responseId"`
	Payload    json.RawMessage `json:"payload"`
}

func (t *transport) execute(ctx context.Context, req apiRequest, out any) error {
	status, body, err := t.send(ctx, req, "application/json")
	if err != nil {
		return err
	}
	payload, err := unwrap(status, body)
	if err != nil {
		return err
	}
	if out == nil || len(payload) == 0 || string(payload) == "null" {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return &Error{Kind: KindAPI, HTTPStatus: status, Code: successCode, Message: "cannot parse Payriff response: " + err.Error(), Err: err}
	}
	return nil
}

func (t *transport) download(ctx context.Context, req apiRequest) ([]byte, error) {
	status, body, err := t.send(ctx, req, "application/pdf, application/json")
	if err != nil {
		return nil, err
	}
	if status >= 200 && status < 300 {
		return body, nil
	}
	if _, err := unwrap(status, body); err != nil {
		return nil, err
	}
	return nil, &Error{Kind: KindAPI, HTTPStatus: status, Message: "unexpected response"}
}

func (t *transport) send(ctx context.Context, req apiRequest, accept string) (int, []byte, error) {
	payload, err := t.requestBody(req)
	if err != nil {
		return 0, nil, err
	}
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.method, t.url(req), reader)
	if err != nil {
		return 0, nil, invalid("%v", err)
	}
	httpReq.Header.Set("Accept", accept)
	httpReq.Header.Set("User-Agent", "payriff-go/"+Version)
	httpReq.Header.Set("Authorization", t.appKey)
	for name, value := range req.headers {
		if value != "" {
			httpReq.Header.Set(name, value)
		}
	}
	if payload != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return 0, nil, &Error{Kind: KindConnection, Message: "request failed: " + err.Error(), Err: err}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, &Error{Kind: KindConnection, Message: "reading response failed: " + err.Error(), Err: err}
	}
	return resp.StatusCode, body, nil
}

func (t *transport) url(req apiRequest) string {
	var parts []string
	for _, kv := range req.query {
		if kv[1] != "" {
			parts = append(parts, queryEscape(kv[0])+"="+queryEscape(kv[1]))
		}
	}
	u := t.baseURL + req.path
	if len(parts) > 0 {
		u += "?" + strings.Join(parts, "&")
	}
	return u
}

func queryEscape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func (t *transport) requestBody(req apiRequest) ([]byte, error) {
	body := req.body
	if req.merchantEnvelope {
		if t.merchantID == "" {
			return nil, invalid("merchant ID must be configured on the client for this operation")
		}
		if body == nil {
			body = map[string]any{}
		}
		body = map[string]any{"merchant": t.merchantID, "body": body}
	}
	if body == nil {
		return nil, nil
	}
	return marshal(body)
}

func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, invalid("request body cannot be serialized: %v", err)
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func unwrap(status int, body []byte) (json.RawMessage, error) {
	ok := status >= 200 && status < 300
	var env envelope
	if len(body) == 0 || json.Unmarshal(body, &env) != nil || env.Code == nil {
		message := "unexpected response from Payriff"
		if !ok {
			message = truncate(body, status)
		}
		return nil, &Error{Kind: KindAPI, HTTPStatus: status, Message: message}
	}
	if !ok || *env.Code != successCode {
		return nil, newAPIError(deref(env.Message), status, *env.Code, deref(env.ResponseID))
	}
	return env.Payload, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func truncate(body []byte, status int) string {
	if len(body) == 0 {
		return fmt.Sprintf("Payriff request failed (HTTP %d)", status)
	}
	if len(body) > maxErrorBody {
		return string(body[:maxErrorBody]) + "..."
	}
	return string(body)
}

func noRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
