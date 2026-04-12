package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GotifyMessage represents a message payload for the Gotify API.
type GotifyMessage struct {
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Extras   Extras `json:"extras,omitempty"`
}

// Extras holds Gotify message extras for client display configuration.
type Extras struct {
	ClientDisplay ClientDisplay `json:"client::display,omitempty"`
}

// ClientDisplay configures how the Gotify client renders the message.
type ClientDisplay struct {
	ContentType string `json:"contentType,omitempty"`
}

// SendMessageArgs are the arguments for the send-message tool.
type SendMessageArgs struct {
	Message     string  `json:"message" jsonschema:"The message content to send"`
	Title       string  `json:"title,omitempty" jsonschema:"Optional title for the message"`
	Priority    float64 `json:"priority,omitempty" jsonschema:"Message priority (0-10, default: 5)"`
	ContentType string  `json:"contentType,omitempty" jsonschema:"Message content type (text/plain or text/markdown, default: text/plain)"`
}

// AskForHelpArgs are the arguments for the ask-for-help tool.
type AskForHelpArgs struct {
	Context string `json:"context" jsonschema:"Context or description of what help is needed"`
	Error   string `json:"error,omitempty" jsonschema:"Optional error message or details"`
}

// NotifyCompletionArgs are the arguments for the notify-completion tool.
type NotifyCompletionArgs struct {
	Task   string `json:"task" jsonschema:"Description of the completed task"`
	Result string `json:"result,omitempty" jsonschema:"Optional result or outcome details"`
}

// SummarizeActivityArgs are the arguments for the summarize-activity tool.
type SummarizeActivityArgs struct {
	Summary string `json:"summary" jsonschema:"Summary of activities or current status"`
	Details string `json:"details,omitempty" jsonschema:"Optional additional details"`
}

// GotifyClient sends messages to a Gotify server.
type GotifyClient struct {
	URL        string
	Token      string
	HTTPClient *http.Client
}

// NewGotifyClientFromEnv creates a GotifyClient from environment variables.
func NewGotifyClientFromEnv() (*GotifyClient, error) {
	url := os.Getenv("GOTIFY_URL")
	token := os.Getenv("GOTIFY_TOKEN")

	if url == "" {
		return nil, fmt.Errorf("GOTIFY_URL environment variable is not set")
	}
	if token == "" {
		return nil, fmt.Errorf("GOTIFY_TOKEN environment variable is not set")
	}

	return &GotifyClient{
		URL:        url,
		Token:      token,
		HTTPClient: http.DefaultClient,
	}, nil
}

// Send posts a GotifyMessage to the server using the provided context.
func (c *GotifyClient) Send(ctx context.Context, msg GotifyMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("%s/message", c.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gotify server returned status: %d", resp.StatusCode)
	}

	return nil
}

// toolResult is a helper that creates a successful or error CallToolResult.
func toolResult(text string, isError bool) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		IsError: isError,
	}, nil, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Gotify Notification Server",
		Version: "1.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "send-message",
		Description: "Send a message to a Gotify server for notifications",
	}, sendMessage)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ask-for-help",
		Description: "Send a help request notification to the user via Gotify",
	}, askForHelp)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "notify-completion",
		Description: "Send a completion notification to the user via Gotify",
	}, notifyCompletion)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "summarize-activity",
		Description: "Send a summary of current activities or status to the user via Gotify",
	}, summarizeActivity)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v\n", err)
	}
}

func sendMessage(ctx context.Context, _ *mcp.CallToolRequest, args SendMessageArgs) (*mcp.CallToolResult, any, error) {
	client, err := NewGotifyClientFromEnv()
	if err != nil {
		return toolResult(fmt.Sprintf("Failed to send message: %s", err), true)
	}

	priority := 5
	if args.Priority > 0 {
		priority = int(args.Priority)
	}

	msg := GotifyMessage{
		Message:  args.Message,
		Title:    args.Title,
		Priority: priority,
	}

	if args.ContentType != "" {
		msg.Extras = Extras{
			ClientDisplay: ClientDisplay{ContentType: args.ContentType},
		}
	}

	if err := client.Send(ctx, msg); err != nil {
		return toolResult(fmt.Sprintf("Failed to send message: %s", err), true)
	}

	return toolResult("Message sent successfully", false)
}

func askForHelp(ctx context.Context, _ *mcp.CallToolRequest, args AskForHelpArgs) (*mcp.CallToolResult, any, error) {
	client, err := NewGotifyClientFromEnv()
	if err != nil {
		return toolResult(fmt.Sprintf("Failed to send help request: %s", err), true)
	}

	message := fmt.Sprintf("🆘 Help needed: %s", args.Context)
	if args.Error != "" {
		message += fmt.Sprintf("\nError: %s", args.Error)
	}

	if err := client.Send(ctx, GotifyMessage{
		Message:  message,
		Title:    "Help Request",
		Priority: 8,
	}); err != nil {
		return toolResult(fmt.Sprintf("Failed to send help request: %s", err), true)
	}

	return toolResult("Help request sent successfully", false)
}

func notifyCompletion(ctx context.Context, _ *mcp.CallToolRequest, args NotifyCompletionArgs) (*mcp.CallToolResult, any, error) {
	client, err := NewGotifyClientFromEnv()
	if err != nil {
		return toolResult(fmt.Sprintf("Failed to send completion notification: %s", err), true)
	}

	message := fmt.Sprintf("✅ Task completed: %s", args.Task)
	if args.Result != "" {
		message += fmt.Sprintf("\nResult: %s", args.Result)
	}

	if err := client.Send(ctx, GotifyMessage{
		Message:  message,
		Title:    "Task Completed",
		Priority: 6,
	}); err != nil {
		return toolResult(fmt.Sprintf("Failed to send completion notification: %s", err), true)
	}

	return toolResult("Completion notification sent successfully", false)
}

func summarizeActivity(ctx context.Context, _ *mcp.CallToolRequest, args SummarizeActivityArgs) (*mcp.CallToolResult, any, error) {
	client, err := NewGotifyClientFromEnv()
	if err != nil {
		return toolResult(fmt.Sprintf("Failed to send summary: %s", err), true)
	}

	message := fmt.Sprintf("📊 Activity Summary: %s", args.Summary)
	if args.Details != "" {
		message += fmt.Sprintf("\nDetails: %s", args.Details)
	}

	if err := client.Send(ctx, GotifyMessage{
		Message:  message,
		Title:    "Activity Summary",
		Priority: 4,
	}); err != nil {
		return toolResult(fmt.Sprintf("Failed to send summary: %s", err), true)
	}

	return toolResult("Activity summary sent successfully", false)
}
