package main

import (
	"context"
	"encoding/json"
	"io"
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

func testPriority(priority float64) *float64 {
	return &priority
}

func decodeRequestBody(t *testing.T, r *http.Request, dest any) {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		t.Errorf("failed to decode body: %v", err)
	}
}

func TestGotifyClient_Send(t *testing.T) {
	var received GotifyMessage
	var requestPath string

	client, ts := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-Gotify-Key") != "test-token" {
			t.Errorf("expected X-Gotify-Key test-token, got %s", r.Header.Get("X-Gotify-Key"))
		}
		decodeRequestBody(t, r, &received)
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	msg := GotifyMessage{
		Message:  "hello",
		Title:    "Test",
		Priority: 5,
	}
	if err := client.Send(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Message != "hello" {
		t.Errorf("expected message 'hello', got %q", received.Message)
	}
	if received.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", received.Title)
	}
	if requestPath != "/message" {
		t.Errorf("expected request path /message, got %q", requestPath)
	}
}

func TestGotifyClient_Send_ServerError(t *testing.T) {
	client, ts := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer ts.Close()

	err := client.Send(context.Background(), GotifyMessage{Message: "test"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGotifyClient_Send_WithExtras(t *testing.T) {
	var received GotifyMessage

	client, ts := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
		w.WriteHeader(http.StatusOK)
	})
	client.URL = ts.URL + "/"
	defer ts.Close()

	msg := GotifyMessage{
		Message:  "markdown msg",
		Priority: 5,
		Extras: &Extras{
			ClientDisplay: ClientDisplay{ContentType: "text/markdown"},
		},
	}
	if err := client.Send(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Extras.ClientDisplay.ContentType != "text/markdown" {
		t.Errorf("expected content type text/markdown, got %q", received.Extras.ClientDisplay.ContentType)
	}
}

func TestGotifyClient_Send_DefaultsNilHTTPClient(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &GotifyClient{
		URL:   ts.URL,
		Token: "test-token",
	}

	err := client.Send(context.Background(), GotifyMessage{
		Message:  "hello",
		Priority: defaultPriority,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Message != "hello" {
		t.Errorf("expected message 'hello', got %q", received.Message)
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
		decodeRequestBody(t, r, &received)
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
	if received.Priority != defaultPriority {
		t.Errorf("expected default priority %d, got %d", defaultPriority, received.Priority)
	}
}

func TestSendMessage_ExplicitZeroPriority(t *testing.T) {
	var received GotifyMessage
	var raw map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if err := json.Unmarshal(body, &received); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatalf("failed to decode raw body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{
		Message:  "low priority",
		Priority: testPriority(0),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected success")
	}
	if received.Priority != 0 {
		t.Errorf("expected explicit priority 0, got %d", received.Priority)
	}
	if _, ok := raw["priority"]; !ok {
		t.Error("expected priority field to be present in request body")
	}
	if _, ok := raw["extras"]; ok {
		t.Error("expected extras to be omitted when no content type is set")
	}
}

func TestSendMessage_CustomPriorityAndContentType(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{
		Message:     "md test",
		Priority:    testPriority(8),
		ContentType: contentTypeMD,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected success")
	}

	if received.Priority != 8 {
		t.Errorf("expected priority 8, got %d", received.Priority)
	}
	if received.Extras.ClientDisplay.ContentType != contentTypeMD {
		t.Errorf("expected %s, got %q", contentTypeMD, received.Extras.ClientDisplay.ContentType)
	}
}

func TestSendMessage_TrimsContentType(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{
		Message:     "md test",
		ContentType: "  " + contentTypeMD + "  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if received.Extras.ClientDisplay.ContentType != contentTypeMD {
		t.Errorf("expected trimmed content type %s, got %q", contentTypeMD, received.Extras.ClientDisplay.ContentType)
	}
}

func TestAskForHelp(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
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
		decodeRequestBody(t, r, &received)
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

func TestSendMessage_InvalidPriority(t *testing.T) {
	t.Setenv("GOTIFY_URL", "http://localhost")
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{
		Message:  "test",
		Priority: testPriority(11),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for out-of-range priority")
	}
}

func TestSendMessage_FractionalPriority(t *testing.T) {
	t.Setenv("GOTIFY_URL", "http://localhost")
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{
		Message:  "test",
		Priority: testPriority(4.5),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for fractional priority")
	}
}

func TestSendMessage_InvalidContentType(t *testing.T) {
	t.Setenv("GOTIFY_URL", "http://localhost")
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{
		Message:     "test",
		ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid content type")
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
		decodeRequestBody(t, r, &received)
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
		decodeRequestBody(t, r, &received)
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
		decodeRequestBody(t, r, &received)
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
	err := client.Send(context.Background(), GotifyMessage{Message: "test"})
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}
func TestGotifyClient_Send_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &GotifyClient{
		URL:        ts.URL,
		Token:      "tok",
		HTTPClient: ts.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Send(ctx, GotifyMessage{Message: "test"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestGotifyClient_Send_UsesHeader(t *testing.T) {
	var gotHeader string
	var gotQuery string

	client, ts := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Gotify-Key")
		gotQuery = r.URL.Query().Get("token")
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	if err := client.Send(context.Background(), GotifyMessage{Message: "test"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHeader != "test-token" {
		t.Errorf("expected X-Gotify-Key header 'test-token', got %q", gotHeader)
	}
	if gotQuery != "" {
		t.Error("token should not be in query params")
	}
}

func TestSummarizeActivity(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
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

func TestSendMessage_TrimsMessageAndTitle(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{
		Message: "  hello world  ",
		Title:   "  Heads up  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if received.Message != "hello world" {
		t.Errorf("expected trimmed message, got %q", received.Message)
	}
	if received.Title != "Heads up" {
		t.Errorf("expected trimmed title, got %q", received.Title)
	}
}

func TestSendMessage_RejectsBlankMessage(t *testing.T) {
	t.Setenv("GOTIFY_URL", "http://localhost")
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := sendMessage(context.Background(), nil, SendMessageArgs{Message: " \n\t "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for blank message")
	}
}

func TestAskForHelp_TrimsFields(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := askForHelp(context.Background(), nil, AskForHelpArgs{
		Context: "  stuck on deploy  ",
		Error:   "  timeout  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if received.Message != "🆘 Help needed: stuck on deploy\nError: timeout" {
		t.Errorf("unexpected trimmed message: %q", received.Message)
	}
}

func TestAskForHelp_RejectsBlankContext(t *testing.T) {
	t.Setenv("GOTIFY_URL", "http://localhost")
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := askForHelp(context.Background(), nil, AskForHelpArgs{Context: "   "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for blank context")
	}
}

func TestNotifyCompletion_TrimsFields(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := notifyCompletion(context.Background(), nil, NotifyCompletionArgs{
		Task:   "  build  ",
		Result: "  success  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if received.Message != "✅ Task completed: build\nResult: success" {
		t.Errorf("unexpected trimmed message: %q", received.Message)
	}
}

func TestNotifyCompletion_RejectsBlankTask(t *testing.T) {
	t.Setenv("GOTIFY_URL", "http://localhost")
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := notifyCompletion(context.Background(), nil, NotifyCompletionArgs{Task: "\t\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for blank task")
	}
}

func TestSummarizeActivity_TrimsFields(t *testing.T) {
	var received GotifyMessage

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequestBody(t, r, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Setenv("GOTIFY_URL", ts.URL)
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := summarizeActivity(context.Background(), nil, SummarizeActivityArgs{
		Summary: "  all good  ",
		Details: "  shipped fix  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if received.Message != "📊 Activity Summary: all good\nDetails: shipped fix" {
		t.Errorf("unexpected trimmed message: %q", received.Message)
	}
}

func TestSummarizeActivity_RejectsBlankSummary(t *testing.T) {
	t.Setenv("GOTIFY_URL", "http://localhost")
	t.Setenv("GOTIFY_TOKEN", "tok")

	result, _, err := summarizeActivity(context.Background(), nil, SummarizeActivityArgs{Summary: "   "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for blank summary")
	}
}
