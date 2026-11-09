# Flow Editor architecture

Flow Editor is a single-window [NSGo](https://github.com/oneo-me/nsgo) desktop app. The editor holds one document, an app's screens each with a design and its flow charts, with its undo history; the sidebar, design and diagram draw it, and the assistant runs a model conversation whose tool calls edit that document.

## Code map

| Path | Responsibility |
| --- | --- |
| `flowfile/` | The flow file format: screens, interface elements and flows, parsing, canonical formatting and edits. Its package comment is the format's reference. |
| `agent/` | The OpenAI-compatible client, the drawing and folder tools and the system prompt, with no UI. |
| `main.go` | The window, the menu and the wiring between editor and assistant. |
| `editor.go`, `diagram.go` | The open document, its history, files and open screen and flow, and the flow chart. |
| `screens.go`, `mockup.go`, `icons.go` | The screen list, the workspace's tabs and the drawing of a screen's design with NSGo controls. |
| `assistant.go` | The conversation and the turns that run the model. |
| `settings.go` | The provider and statistics settings. |
| `stats.go` | The opt-out usage statistics. |
| `locales/` | Messages in `en.yaml` and `zh-Hans.yaml`; `catalog_gen.go` is generated. |

## Boundaries

- `flowfile` and `agent` never import NSGo, so the format and the model client stay testable on their own.
- Every change to the document goes through the editor, which records it for undo. Tool calls of one assistant turn undo together.
- A design is drawn with NSGo's own controls, which keep their own state so it can be tried. A tablet or desktop design narrower than its device lays itself out anew instead of scaling: macOS leaves out what a scaled view's frame would have outside the window.
- The flow file names icons, as `flowfile` cannot import NSGo; `icons.go` maps each name to its Lucide icon, and a test keeps the two lists equal.
- One flow view follows the open flow instead of remounting per flow, and the design covers it while its tab is open, and fits the chart only once a change's batch is over: remounting a chart, or fitting it while it takes new data, trips NSGo's commit check on macOS.
- A conversation belongs to the document it began on; opening or creating another document clears it.
- The assistant only reads the attached project folder, through an `os.Root` that symbolic links cannot leave; folder tools are offered only while one is attached.
- User-visible words live in `locales/`, in both languages.
- Statistics never carry document content, messages, paths or keys. `nsgo dev` builds and test binaries send none.

## Core flows

**Assistant turn.** Sending a message starts a turn on a goroutine that streams the model's reply. Each tool call is applied on the UI's goroutine through the editor, which then opens the screen and flow the call touched, and its result goes back to the model until it answers in text. Tools that omit a screen or flow act on the open one. Folder tools read files on the turn's goroutine instead, so a large project never blocks the UI.

**Files.** Opening parses a flow file and replaces the document with a fresh history; a file without screens, from before screens existed, reads as one screen with one flow. Saving writes the canonical form, so comments in a hand-written file are dropped.

**Statistics.** Release builds send an Umami pageview at start, an event per message sent and heartbeats while the chart or the conversation changes. A queue on its own goroutine posts them and drops failures.

## Data and state

Settings, the attached folder's path and the install's random statistics ID live in NSGo's storage on the device. Documents live only in the files the user opens and saves.

## Verification

`go tool nsgo test` drives the app headless against a fake model served by `httptest`, and covers the format and the client. Check layout and platform behavior by hand with `go tool nsgo dev`, or with `go tool nsgo test -target macos`.

## Subjects

| Document | When to read |
| --- | --- |
| [`flowfile` package comment](flowfile/) | Changing the flow file format |
