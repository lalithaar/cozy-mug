---
title: "settings package"
description: "Minimal environment-based configuration."
---

`settings.Load()` reads `DATABASE_URL`, `APP_ENV`/`DEV`, and `PORT`. Defaults
make a fresh app run with zero config. There is no config file by design.
