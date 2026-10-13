# 0019 — auth hardening helpers (lockout and password change)

- Status: firm (implemented)
- Date: 2026-10-13
- Supersedes: none

## Opinion
`auth` now includes two optional, delivery-independent hardening helpers:
`Lockout`, an in-memory repeated-failure tracker, and `ChangePassword`, a
verified password-change primitive. Neither is wired into the framework by
default; both remain app-callable. Reset/verification flows and mail delivery
stay deferred with email/jobs.

## Because
Brute-force resistance (lockout) and password rotation are useful identity
depth without reopening the deferred mail/jobs nutrient. An in-memory limiter
is sufficient for a single-instance app or agent testing, while multi-instance
deployments can replace it with their own storage-backed counter.

## Alternatives rejected
- DB-owned lockout table in core — rejected; it would matter the app's schema and complicate the small app story.
- Built-in password-reset flow — rejected because delivery is blocked by deferred mail.
- Distributed rate limiting by default — rejected; too much state and config for the 90%.

## Apply when
Apps need repeated-failure lockout or verified password change.

## Deviate / ask when
A deployment needs distributed limiting or mail-backed flows; add a store
implementation or revisit the mail deferral rather than bending these helpers.