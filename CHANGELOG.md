# Changelog

## v0.1.1 (unreleased)

- fix(client): include server error body in returned errors; drop no-op `omitempty`
- fix: omit empty extras and align version to 0.1.0
- fix(priority): serialize explicit zero priority correctly; harden CI and add GoReleaser
- test: fail on malformed Gotify request payloads instead of silently ignoring decode errors
- chore: update dependencies (`go-sdk` v1.4.1 → v1.7.0, `golang.org/x/sys`, `golang.org/x/tools`, etc.)
- ci: switch release workflow to GoReleaser Pro

## v0.1.0

- Initial versioned release: Gotify MCP server with send-message, ask-for-help,
  notify-completion, and summarize-activity tools.

## v0.0.1

- Initial release.
