# Flow Editor

A desktop app and flow designer built on [NSGo](https://github.com/oneo-me/nsgo): a flow file's screens and their flow charts on the left, an AI assistant that draws them on the right. Read [ARCHITECTURE.md](ARCHITECTURE.md) before coding.

## Principles

- Follow NSGo's conventions: user-visible words go through `locales/` in both languages, never string literals.
- Tests run against a fake model over `httptest`; never call a real provider from tests.
- Never let usage statistics carry document content, messages, paths or keys, or report from `nsgo dev` builds and tests.

## Verification

- Run `go tool nsgo check` and `go tool nsgo test`, never `go vet`, `go test` or `go build` directly.
- Do not run `go mod tidy`: it drops the requirements `nsgo check` needs for TSX type generation. Update NSGo with `go get github.com/oneo-me/nsgo@<version>`.

## Documentation

- Update a document only when a fact its reader needs changes.
- Avoid secondary labels and descriptions that restate a heading or value; let the layout explain.
- Write only what code and comments cannot show: purpose, boundaries and reasons, not names, fields, defaults or step-by-step behavior.
- Keep each document within its budget; check its size after editing it and trim it before handing off when it is over.
- Record completed user-visible outcomes as one-clause `Unreleased` bullets in [CHANGELOG.md](CHANGELOG.md), ordered by type and merging related ones.
- Repair documents you touch instead of only appending to them.
- Keep `README.md` and `README.zh-CN.md` equivalent; everything else, including code comments, is English.
