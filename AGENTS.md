# Flow Editor

A desktop flow chart editor built on [NSGo](https://github.com/oneo-me/nsgo): a flow file's chart on the left, an AI assistant that draws on it on the right.

## Layout

- `flowfile/`: the flow file format (parse, format, edits). Its package comment is the format's reference.
- `agent/`: the OpenAI-compatible client (streaming, tool calls), the drawing tools and the system prompt.
- `main.go`, `editor.go`, `diagram.go`, `assistant.go`, `settings.go`: the app.
- `locales/`: messages in `en.yaml` and `zh-Hans.yaml`; `catalog_gen.go` is generated.

## Rules

- Follow NSGo's conventions: user-visible words go through `locales/` in both languages, never string literals.
- Run `go tool nsgo check` and `go tool nsgo test`, never `go vet`, `go test` or `go build` directly.
- Do not run `go mod tidy`: it drops the requirements `nsgo check` needs for TSX type generation. Update NSGo with `go get github.com/oneo-me/nsgo@<version>`.
- Tests run against a fake model over `httptest`; never call a real provider from tests.
