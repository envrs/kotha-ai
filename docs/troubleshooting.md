# Troubleshooting

Common issues and their fixes.

## Server won't start

- Check that the port is free: `lsof -i :8080`.
- Verify the config file is valid JSON: `kotha schema validate`.
- Ensure the working directory exists and is readable.

## LSP server not found

- Install the language server for your language, e.g. `go install golang.org/x/tools/gopls@latest`.
- Confirm the `command` in `lsp` config points to an executable on your `PATH`.

## Agent stalls or hangs

- Cancel an in-flight request with `Ctrl+C` in the TUI or `kotha api cancel -s <url> -i <session>`.
- Reduce `maxTokens` or switch to a smaller model if you hit provider rate limits.

## No symbols indexed

- Ensure the LSP server for the language is configured and installed.
- Check `--log-level debug` output for `Symbol indexing failed` warnings.
- Symbol indexing runs after the server becomes ready; large repos may take a while.

## WebSocket streaming fails

- The client must upgrade with `Sec-WebSocket-Key` and `Connection: Upgrade` headers; browsers and `gorilla/websocket` handle this automatically.
- Ensure the server is reachable at `ws://<host>/v1/sessions/{session_id}/ask/ws`.

## Database errors

- A corrupt SQLite file can be restored by deleting it; sessions and messages are recreated on next run.
- Migrations run automatically on startup; the `symbols` migration was added in v0.5.0.

## Getting help

- Run `kotha --help` for CLI usage.
- Run `kotha api --help` for server client commands.
- Report issues at the project repository.