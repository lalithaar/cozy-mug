---
title: "Dev server"
description: "Fast restarts and automatic browser refresh during development."
---

`cozy dev` builds your app into a stable binary (`.cozy/dev.exe`) and watches
the source tree. On a change it rebuilds, restarts the process, and the browser
refreshes itself.

## What it does

1. Polls `.go`, `.html`, `.css`, `.js`, `.tmpl`, `go.mod`, and `go.sum`.
2. Rebuilds to the same binary path.
3. Stops the old process and starts the new one.
4. Streams logs and reports crashes; rebuilds on the next change after a limit.

## Browser refresh

In development, the app injects a script that polls
`/__cozy/dev-build`. When the server restarts and the build id changes, the
page reloads.

These endpoints and the injection exist only when `Config.Dev` is true.
