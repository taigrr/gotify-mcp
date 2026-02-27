package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestServer(t *testing.T, handler http.HandlerFunc) (*GotifyClient, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	return &GotifyClient{
		URL:        ts.URL,
		Token:      "test-token",
		HTTPClient: ts.Client(),
	}, ts
}

func TestGotifyClient_Send(t *testing.T) {
	var received GotifyMessage

	client, ts := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Query().Get("token") != "test-token" {
			t.Errorf("expected token test-token, got %s", r.URL.Query().Get("token"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	msg := GotifyMessage{
		Message:  "hello",
		Title:    "Test",
		Priority: 5,
	}
	if err := client.Send(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Message != "hello" {
		t.Errorf("expected message 'hello', got %q", received.Message)
	}
	if received.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", received.Title)
	}
}

func TestGotifyClient_Send_ServerError(t *testing.T) {
	client, ts := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer ts.Close()

	err := client.Send(GotifyMessage{Message: "test"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGotifyClient_Send_WithExtras(t *testing.T) {
	var received GotifyMessage

	client, ts := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	msg := GotifyMessage{
		Message:  "markdown msg",
		Priority: 5,
		Extras: Extras{
			ClientDisplay: ClientDisplay{ContentType: "text/markdown"},
		},
	}
	if err := client.Send(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Extras.ClientDisplay.ContentType != "text/markdown" {
		t.Errorf("expected content type text/markdown, got %q", received.Extras.ClientDisplay.ContentType)
	}
}

func TestNewGotifyClientFromEnv_MissingURL(t *testing.T) {
	t.Setenv("GOTIFY_URL", "")
	t.Setenv("GOTIFY_TOKEN", "tok")
	_, err := NewGotifyClientFromEnv()
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestNewGotifyClientFromEnv_MissingToken(t *testing.T) {
	t.Setenv("GOTIFY_URL", "http://localhost")
	t.Setenv("GOTIFY_TOKEN", "")
	_, err := NewGotifyClientFromEnv()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestToolResult(t *testing.T) {
	result, extra, err := toolResult("ok", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extra != nil {
		t.Errorf("expected nil extra, got %v", extra)
	}
	if result.IsError {
		t.Error("expected IsError to be false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if tc.Text != "ok" {
		t.Errorf("expected 'ok', got %q", tc.Text)
	}
}

func TestSendMessage_DefaultPriority(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{
		Message: "test msg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected success")
	}
	if received.Priority != 5 {
		t.Errorf("expected default priority 5, got %d", received.Priority)
	}
}

func TestSendMessage_CustomPriorityAndContentType(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	sendMessage(context.Background(), nil, SendMessageArgs{
		Message:     "md test",
		Priority:    8,
		ContentType: "text/markdown",
	})

	if received.Priority != 8 {
		t.Errorf("expected priority 8, got %d", received.Priority)
	}
	if received.Extras.ClientDisplay.ContentType != "text/markdown" {
		t.Errorf("expected text/markdown, got %q", received.Extras.ClientDisplay.ContentType)
	}
}

func TestAskForHelp(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, _ := askForHelp(context.Background(), nil, AskForHelpArgs{
		Context: "stuck on deploy",
		Error:   "timeout",
	})
	if result.IsError {
		t.Error("expected success")
	}
	if received.Priority != 8 {
		t.Errorf("expected priority 8, got %d", received.Priority)
	}
	if received.Title != "Help Request" {
		t.Errorf("expected title 'Help Request', got %q", received.Title)
	}
}

func TestNotifyCompletion(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, _ := notifyCompletion(context.Background(), nil, NotifyCompletionArgs{
		Task:   "build",
		Result: "success",
	})
	if result.IsError {
		t.Error("expected success")
	}
	if received.Priority != 6 {
		t.Errorf("expected priority 6, got %d", received.Priority)
	}
}

func TestSendMessage_MissingEnv(t *testing.T) {
	t.Setenv("GOTIFY_URL", "")
	t.Setenv("GOTIFY_TOKEN", "")
	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{Message: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing env")
	}
}

func TestAskForHelp_MissingEnv(t *testing.T) {
	t.Setenv("GOTIFY_URL", "")
	t.Setenv("GOTIFY_TOKEN", "")
	result, _, err := askForHelp(context.Background(), nil, AskForHelpArgs{Context: "help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing env")
	}
}

func TestNotifyCompletion_MissingEnv(t *testing.T) {
	t.Setenv("GOTIFY_URL", "")
	t.Setenv("GOTIFY_TOKEN", "")
	result, _, err := notifyCompletion(context.Background(), nil, NotifyCompletionArgs{Task: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing env")
	}
}

func TestSummarizeActivity_MissingEnv(t *testing.T) {
	t.Setenv("GOTIFY_URL", "")
	t.Setenv("GOTIFY_TOKEN", "")
	result, _, err := summarizeActivity(context.Background(), nil, SummarizeActivityArgs{Summary: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing env")
	}
}

func TestSendMessage_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")
	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{Message: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for server error")
	}
}

func TestAskForHelp_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")
	result, _, err := askForHelp(context.Background(), nil, AskForHelpArgs{Context: "help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for server error")
	}
}

func TestNotifyCompletion_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")
	result, _, err := notifyCompletion(context.Background(), nil, NotifyCompletionArgs{Task: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for server error")
	}
}

func TestSummarizeActivity_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")
	result, _, err := summarizeActivity(context.Background(), nil, SummarizeActivityArgs{Summary: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for server error")
	}
}

func TestAskForHelp_WithoutError(t *testing.T) {
	var received GotifyMessage
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")
	result, _, _ := askForHelp(context.Background(), nil, AskForHelpArgs{Context: "stuck"})
	if result.IsError {
		t.Error("expected success")
	}
	if received.Message != "🆘 Help needed: stuck" {
		t.Errorf("unexpected message: %q", received.Message)
	}
}

func TestNotifyCompletion_WithoutResult(t *testing.T) {
	var received GotifyMessage
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")
	result, _, _ := notifyCompletion(context.Background(), nil, NotifyCompletionArgs{Task: "build"})
	if result.IsError {
		t.Error("expected success")
	}
	if received.Message != "✅ Task completed: build" {
		t.Errorf("unexpected message: %q", received.Message)
	}
}

func TestSummarizeActivity_WithoutDetails(t *testing.T) {
	var received GotifyMessage
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")
	result, _, _ := summarizeActivity(context.Background(), nil, SummarizeActivityArgs{Summary: "all good"})
	if result.IsError {
		t.Error("expected success")
	}
	if received.Message != "📊 Activity Summary: all good" {
		t.Errorf("unexpected message: %q", received.Message)
	}
}

func TestGotifyClient_Send_ConnectionError(t *testing.T) {
	client := &GotifyClient{
		URL:        "http://127.0.0.1:1",
		Token:      "tok",
		HTTPClient: http.DefaultClient,
	}
	err := client.Send(GotifyMessage{Message: "test"})
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}
func TestSummarizeActivity(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, _ := summarizeActivity(context.Background(), nil, SummarizeActivityArgs{
		Summary: "all good",
		Details: "3 tasks done",
	})
	if result.IsError {
		t.Error("expected success")
	}
	if received.Priority != 4 {
		t.Errorf("expected priority 4, got %d", received.Priority)
	}
}
