# gotify-mcp

[![License 0BSD](https://img.shields.io/badge/License-0BSD-pink.svg)](https://opensource.org/licenses/0BSD)
[![GoDoc](https://godoc.org/github.com/taigrr/gotify-mcp?status.svg)](https://godoc.org/github.com/taigrr/gotify-mcp)
[![Go Mod](https://img.shields.io/badge/go.mod-v1.23-blue)](go.mod)

A Model Context Protocol (MCP) server that enables LLMs to send notifications to a Gotify server.
This allows AI assistants to notify users about task completion, request help, or provide activity summaries.

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

```bash
go install github.com/taigrr/gotify-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/taigrr/gotify-mcp.git
cd gotify-mcp
go build -o gotify-mcp
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

This project is licensed under the 0BSD License, written by [Rob Landley](https://github.com/landley).
As such, you may use this library without restriction or attribution, but please don't pass it off as your own.
Attribution, though not required, is appreciated.

By contributing, you agree all code submitted also falls under the License.