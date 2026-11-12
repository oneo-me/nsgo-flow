# Flow Editor architecture

Flow Editor is a single-window [NSGo](https://github.com/nsgo-dev/nsgo) desktop app, for desktops only. The editor holds the open documents, a tab each across the window's top, beside the assistant running the window's height; a document is an app's screens each with a design and its flow charts, with its file, undo history and conversation. The sidebar, design and diagram draw the document shown, and the assistant runs a model conversation whose tool calls edit the document it began on.

## Code map

| Path | Responsibility |
| --- | --- |
| `flowfile/` | The flow file format: screens, interface elements and flows, parsing, canonical formatting and edits. Its package comment is the format's reference. |
| `agent/` | The OpenAI-compatible client, the drawing and folder tools and the system prompt, with no UI. |
| `main.go`, `tabs.go` | The window, its title bar of tabs, the menu and the wiring between editor and assistant. |
| `editor.go`, `diagram.go` | The open documents, their history, files and open screen and flow, and the flow chart. |
| `screens.go`, `mockup.go`, `icons.go` | The sidebar of designs and flows, and the drawing of a screen's design. |
| `assistant.go` | The conversation and the turns that run the model. |
| `settings.go` | The provider and statistics settings. |
| `stats.go` | The opt-out usage statistics. |
| `locales/` | Messages in `en.yaml` and `zh-Hans.yaml`; `catalog_gen.go` is generated. |

## Boundaries

- `flowfile` and `agent` never import NSGo, so the format and the model client stay testable on their own.
- Every change to the document goes through the editor, which records it for undo. Tool calls of one assistant turn undo together.
- A design is a draft drawn from a few primitives shaped by Tailwind CSS classes, not from ready-made controls, so the model can draw any look. Classes NSGo does not know go back to the model as a warning with the tool's result. A design lays out at its device's size on a plane the design canvas moves and scales, as the flow chart's camera does, since NSGo's zooming scroll views exist only on Apple platforms. Platforms hit-test views without that transform, so a layer over the design finds the element under the pointer from the elements' frames, as the flow chart finds its nodes. Elements are named by their paths, which the selection and `set_element` use.
- The flow file names icons by their Lucide names, or older ones it maps to them, as `flowfile` cannot import NSGo. `tools/iconsgen`, run by `go generate` after updating NSGo, writes every name NSGo's Lucide package has into `flowfile` and the map from names to icons into the app.
- One flow view follows the open flow of the document shown instead of remounting per flow or tab, and the design covers it while its tab is open, and fits the chart only once a change's batch is over: remounting a chart, or fitting it while it takes new data, trips NSGo's commit check on macOS.
- Each document has its own conversation; a turn keeps editing the document it began on while another tab shows.
- The assistant reads only the folder holding the open file, through an `os.Root` that symbolic links cannot leave, and the other open documents, which the app answers `get_project` with.
- User-visible words live in `locales/`, in both languages.
- Statistics never carry document content, messages, paths or keys. `nsgo dev` builds and test binaries send none.

## Core flows

**Assistant turn.** Sending a message starts a turn on a goroutine that streams the model's reply. Each tool call is applied on the UI's goroutine through the editor, which then opens the screen and flow the call touched, and its result goes back to the model until it answers in text. Tools that omit a screen or flow act on the open one. Folder tools read files on the turn's goroutine instead, so a large project never blocks the UI.

**Files.** Every document has a file: making one asks where to save it first, and a message sent with none open makes one. Opening parses a flow file into a tab of its own, or shows the file's tab when it is open already; a file without screens, from before screens existed, reads as one screen with one flow. A document saves to the path it was opened from or saved to, which NSGo's save dialog reports. Saving writes the canonical form, so comments in a hand-written file are dropped.

**Statistics.** Release builds send an Umami pageview at start, an event per message sent and heartbeats while the chart or the conversation changes. A queue on its own goroutine posts them and drops failures.

## Data and state

Settings, the paths of the files open in tabs and the install's random statistics ID live in NSGo's storage on the device. Documents live only in the files the user opens and saves.

## Verification

`go tool nsgo test` drives the app headless against a fake model served by `httptest`, and covers the format and the client. Check layout and platform behavior by hand with `go tool nsgo dev`, or with `go tool nsgo test -target macos`.

## Subjects

| Document | When to read |
| --- | --- |
| [`flowfile` package comment](flowfile/) | Changing the flow file format |
