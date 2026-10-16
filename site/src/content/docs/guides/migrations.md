---
title: "Migrations"
description: "Generate, review, and apply SQL migrations."
---

cozy-mug never changes schema at boot. Generate a migration from your models:

```bash
cozy migrate new init .
```

This writes `migrations/sqlite/0001_init.sql` and the Postgres variant. Review
the SQL, then apply:

```bash
cozy migrate up
```

`cozy migrate up` records applied files in `schema_migrations`. Raw SQL is
allowed, so destructive changes remain explicit and reviewable.
