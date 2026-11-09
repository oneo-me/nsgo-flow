# Flow Editor

A flow chart editor where you design the chart by talking with an AI. Open a `.flow` file, see it as a diagram on the left, and discuss the process with an assistant on the right, which draws on the chart with tools as you agree.

## Use

```sh
go tool nsgo dev            # run
go tool nsgo build          # build macOS, Windows and Linux into dist/
```

Open **Settings** (⌘,) and enter an OpenAI-compatible provider: its base URL (such as `https://api.openai.com/v1`), API key and a model that supports tool calls. A flow file given on the command line opens at start.

## Flow files

```
title "Order handling"
direction down                  # or right

node start "Order placed" kind=start
node check "Check stock" kind=decision note="Reserve every item"
node pay "Charge card" status=doing

start -> check
check -> pay "in stock"
check ..> start "retry"         # dashed edge
```

Kinds: `task`, `start`, `end`, `decision`, `data`, `note`. Statuses: `todo`, `doing`, `done`, `blocked`, `skipped`. An edge may name undeclared nodes. Saving writes the canonical form without comments. See `examples/order.flow`.
