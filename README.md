# Gotify MCP Server

A Model Context Protocol (MCP) server that enables LLMs to send notifications to a Gotify server. This allows AI assistants to notify users about task completion, request help, or provide activity summaries.

## Features

- **Send Message**: Send custom messages with configurable priority and title
- **Ask for Help**: Send help request notifications with context and error details
- **Notify Completion**: Send task completion notifications with results
- **Summarize Activity**: Send activity summaries with optional details

## Environment Variables

The MCP server requires the following environment variables:

- `GOTIFY_URL`: The URL of your Gotify server (e.g., `https://gotify.example.com`)
- `GOTIFY_TOKEN`: Your Gotify application token for authentication

## Installation

1. Clone the repository:
```bash
git clone https://github.com/taigrr/gotify-mcp.git
cd gotify-mcp
```

2. Build the binary:
```bash
go build -o gotify-mcp
```

3. Set up environment variables:
```bash
export GOTIFY_URL="https://your-gotify-server.com"
export GOTIFY_TOKEN="your-application-token"
```

## Usage

The MCP server communicates over stdio and provides the following tools:

### send-message
Send a custom message to Gotify.

**Parameters:**
- `message` (required): The message content to send
- `title` (optional): Title for the message
- `priority` (optional): Message priority (0-10, default: 5)

### ask-for-help
Send a help request notification.

**Parameters:**
- `context` (required): Context or description of what help is needed
- `error` (optional): Optional error message or details

### notify-completion
Send a task completion notification.

**Parameters:**
- `task` (required): Description of the completed task
- `result` (optional): Optional result or outcome details

### summarize-activity
Send an activity summary notification.

**Parameters:**
- `summary` (required): Summary of activities or current status
- `details` (optional): Optional additional details

## Integration with MCP Clients

To use this MCP server with an MCP client, configure it to run the `gotify-mcp` binary with the appropriate environment variables set.

Example configuration for Claude Desktop:

```json
{
  "mcpServers": {
    "gotify": {
      "command": "/path/to/gotify-mcp",
      "env": {
        "GOTIFY_URL": "https://your-gotify-server.com",
        "GOTIFY_TOKEN": "your-application-token"
      }
    }
  }
}
```

## License

This project is licensed under the MIT License.