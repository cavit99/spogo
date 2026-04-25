package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	ErrNoContent   = errors.New("no content")
	ErrUnsupported = errors.New("unsupported operation")
)

type APIError struct {
	Status  int
	Message string
	Body    string
}

func (e APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("spotify api error (%d): %s", e.Status, e.Message)
	}
	if e.Status != 0 {
		return fmt.Sprintf("spotify api error (%d)", e.Status)
	}
	return "spotify api error"
}

func apiErrorFromResponse(resp *http.Response) error {
	if resp == nil {
		return APIError{Message: "nil response"}
	}
	body, _ := io.ReadAll(resp.Body)
	payload := struct {
		Error struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}{
		Message: resp.Status,
	}
	_ = json.Unmarshal(body, &payload)
	status := resp.StatusCode
	message := payload.Error.Message
	if message == "" {
		message = payload.Message
	}
	if status == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if message == "" || message == resp.Status {
				message = fmt.Sprintf("rate limit exceeded (Retry-After: %s)", retryAfter)
			} else {
				message = fmt.Sprintf("%s (Retry-After: %s)", message, retryAfter)
			}
		}
		bodyExcerpt := string(body)
		if len(bodyExcerpt) > 240 {
			bodyExcerpt = bodyExcerpt[:240] + "…"
		}
		if bodyExcerpt != "" {
			message = fmt.Sprintf("%s :: body=%q", message, bodyExcerpt)
		}
	}
	return APIError{Status: status, Message: message, Body: string(body)}
}
