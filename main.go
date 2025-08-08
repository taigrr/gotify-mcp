package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type GotifyMessage struct {
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

func getStringArg(args map[string]any, key string, defaultValue string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return defaultValue
}

func getNumberArg(args map[string]any, key string, defaultValue float64) float64 {
	if val, ok := args[key].(float64); ok {
		return val
	}
	return defaultValue
}

func main() {
	s := server.NewMCPServer(
		"Gotify Notification Server",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	sendMessageTool := mcp.NewTool("send-message",
		mcp.WithDescription("Send a message to a Gotify server for notifications"),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("The message content to send"),
		),
		mcp.WithString("title",
			mcp.Description("Optional title for the message"),
		),
		mcp.WithNumber("priority",
			mcp.Description("Message priority (0-10, default: 5)"),
		),
	)

	askForHelpTool := mcp.NewTool("ask-for-help",
		mcp.WithDescription("Send a help request notification to the user via Gotify"),
		mcp.WithString("context",
			mcp.Required(),
			mcp.Description("Context or description of what help is needed"),
		),
		mcp.WithString("error",
			mcp.Description("Optional error message or details"),
		),
	)

	notifyCompletionTool := mcp.NewTool("notify-completion",
		mcp.WithDescription("Send a completion notification to the user via Gotify"),
		mcp.WithString("task",
			mcp.Required(),
			mcp.Description("Description of the completed task"),
		),
		mcp.WithString("result",
			mcp.Description("Optional result or outcome details"),
		),
	)

	summarizeTool := mcp.NewTool("summarize-activity",
		mcp.WithDescription("Send a summary of current activities or status to the user via Gotify"),
		mcp.WithString("summary",
			mcp.Required(),
			mcp.Description("Summary of activities or current status"),
		),
		mcp.WithString("details",
			mcp.Description("Optional additional details"),
		),
	)

	s.AddTool(sendMessageTool, sendMessage)
	s.AddTool(askForHelpTool, askForHelp)
	s.AddTool(notifyCompletionTool, notifyCompletion)
	s.AddTool(summarizeTool, summarizeActivity)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
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

func sendMessage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	message, err := request.RequireString("message")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	title := getStringArg(args, "title", "")
	priority := getNumberArg(args, "priority", 5)

	gotifyMsg := GotifyMessage{
		Message:  message,
		Title:    title,
		Priority: int(priority),
	}

	if err := sendGotifyMessage(gotifyMsg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send message: %s", err)), nil
	}

	return mcp.NewToolResultText("Message sent successfully"), nil
}

func askForHelp(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	contextStr, err := request.RequireString("context")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	errorMsg := getStringArg(args, "error", "")

	message := fmt.Sprintf("🆘 Help needed: %s", contextStr)
	if errorMsg != "" {
		message += fmt.Sprintf("\nError: %s", errorMsg)
	}

	gotifyMsg := GotifyMessage{
		Message:  message,
		Title:    "Help Request",
		Priority: 8,
	}

	if err := sendGotifyMessage(gotifyMsg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send help request: %s", err)), nil
	}

	return mcp.NewToolResultText("Help request sent successfully"), nil
}

func notifyCompletion(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	task, err := request.RequireString("task")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	result := getStringArg(args, "result", "")

	message := fmt.Sprintf("✅ Task completed: %s", task)
	if result != "" {
		message += fmt.Sprintf("\nResult: %s", result)
	}

	gotifyMsg := GotifyMessage{
		Message:  message,
		Title:    "Task Completed",
		Priority: 6,
	}

	if err := sendGotifyMessage(gotifyMsg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send completion notification: %s", err)), nil
	}

	return mcp.NewToolResultText("Completion notification sent successfully"), nil
}

func summarizeActivity(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	summary, err := request.RequireString("summary")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	details := getStringArg(args, "details", "")

	message := fmt.Sprintf("📊 Activity Summary: %s", summary)
	if details != "" {
		message += fmt.Sprintf("\nDetails: %s", details)
	}

	gotifyMsg := GotifyMessage{
		Message:  message,
		Title:    "Activity Summary",
		Priority: 4,
	}

	if err := sendGotifyMessage(gotifyMsg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send summary: %s", err)), nil
	}

	return mcp.NewToolResultText("Activity summary sent successfully"), nil
}

