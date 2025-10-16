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

type GotifyMessage struct {
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

type SendMessageArgs struct {
	Message  string  `json:"message" jsonschema:"The message content to send"`
	Title    string  `json:"title,omitempty" jsonschema:"Optional title for the message"`
	Priority float64 `json:"priority,omitempty" jsonschema:"Message priority (0-10 default: 5)"`
}

type AskForHelpArgs struct {
	Context string `json:"context" jsonschema:"Context or description of what help is needed"`
	Error   string `json:"error,omitempty" jsonschema:"Optional error message or details"`
}

type NotifyCompletionArgs struct {
	Task   string `json:"task" jsonschema:"Description of the completed task"`
	Result string `json:"result,omitempty" jsonschema:"Optional result or outcome details"`
}

type SummarizeActivityArgs struct {
	Summary string `json:"summary" jsonschema:"Summary of activities or current status"`
	Details string `json:"details,omitempty" jsonschema:"Optional additional details"`
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Gotify Notification Server",
		Version: "1.0.0",
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

func sendGotifyMessage(message GotifyMessage) error {
	gotifyURL := os.Getenv("GOTIFY_URL")
	gotifyToken := os.Getenv("GOTIFY_TOKEN")

	if gotifyURL == "" {
		return fmt.Errorf("GOTIFY_URL environment variable is not set")
	}
	if gotifyToken == "" {
		return fmt.Errorf("GOTIFY_TOKEN environment variable is not set")
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("%s/message?token=%s", gotifyURL, gotifyToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gotify server returned status: %d", resp.StatusCode)
	}

	return nil
}

func sendMessage(ctx context.Context, req *mcp.CallToolRequest, args SendMessageArgs) (*mcp.CallToolResult, any, error) {
	priority := 5
	if args.Priority > 0 {
		priority = int(args.Priority)
	}

	gotifyMsg := GotifyMessage{
		Message:  args.Message,
		Title:    args.Title,
		Priority: priority,
	}

	if err := sendGotifyMessage(gotifyMsg); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to send message: %s", err)},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Message sent successfully"},
		},
	}, nil, nil
}

func askForHelp(ctx context.Context, req *mcp.CallToolRequest, args AskForHelpArgs) (*mcp.CallToolResult, any, error) {
	message := fmt.Sprintf("🆘 Help needed: %s", args.Context)
	if args.Error != "" {
		message += fmt.Sprintf("\nError: %s", args.Error)
	}

	gotifyMsg := GotifyMessage{
		Message:  message,
		Title:    "Help Request",
		Priority: 8,
	}

	if err := sendGotifyMessage(gotifyMsg); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to send help request: %s", err)},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Help request sent successfully"},
		},
	}, nil, nil
}

func notifyCompletion(ctx context.Context, req *mcp.CallToolRequest, args NotifyCompletionArgs) (*mcp.CallToolResult, any, error) {
	message := fmt.Sprintf("✅ Task completed: %s", args.Task)
	if args.Result != "" {
		message += fmt.Sprintf("\nResult: %s", args.Result)
	}

	gotifyMsg := GotifyMessage{
		Message:  message,
		Title:    "Task Completed",
		Priority: 6,
	}

	if err := sendGotifyMessage(gotifyMsg); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to send completion notification: %s", err)},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Completion notification sent successfully"},
		},
	}, nil, nil
}

func summarizeActivity(ctx context.Context, req *mcp.CallToolRequest, args SummarizeActivityArgs) (*mcp.CallToolResult, any, error) {
	message := fmt.Sprintf("📊 Activity Summary: %s", args.Summary)
	if args.Details != "" {
		message += fmt.Sprintf("\nDetails: %s", args.Details)
	}

	gotifyMsg := GotifyMessage{
		Message:  message,
		Title:    "Activity Summary",
		Priority: 4,
	}

	if err := sendGotifyMessage(gotifyMsg); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to send summary: %s", err)},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Activity summary sent successfully"},
		},
	}, nil, nil
}
