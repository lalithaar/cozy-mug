---
title: "Deployment"
description: "Build and run a cozy-mug app in production."
---

Build the binary:

```bash
go build -o app .
```

Set `APP_ENV=production` (or `DEV=0`) to disable dev pages and secure cookies.
Set `DATABASE_URL` for your database. Put the binary behind TLS and a reverse
proxy. `app.ListenAndServe` sets read/header/write/idle timeouts and graceful
shutdown.

Run migrations before starting the new binary:

```bash
cozy migrate up
```
