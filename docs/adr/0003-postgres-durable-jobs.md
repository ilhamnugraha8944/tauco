# ADR-0003: PostgreSQL sebagai Source of Truth Durable Job

- Status: Accepted
- Tanggal: 28 Juli 2026
- Pemilik: Product / Engineering

## Context

Email notification, activity log, dan image processing tidak boleh memblokir
response HTTP. Goroutine fire-and-forget tidak aman pada runtime scale-to-zero
atau serverless karena process/CPU dapat berhenti setelah response.

External queue saja juga tidak dapat berada dalam transaction yang sama dengan
contact message atau media record.

## Decision

- Simpan `background_jobs` dalam PostgreSQL.
- Buat entity dan job dalam satu database transaction.
- Payload job hanya memuat entity ID, tanpa contact PII.
- Claim memakai `FOR UPDATE SKIP LOCKED`.
- Gunakan lease, heartbeat, bounded goroutine pool, retry exponential dengan
  jitter, dead-letter, replay, dan graceful shutdown.
- Handler bersifat idempotent dan menerima at-least-once delivery.
- External task/scheduler hanya membangunkan worker; bukan source of truth.
- Reconciler mengambil pending job yang gagal dibangunkan.

## Consequences

- Contact message tetap aman walau email/provider wake-up gagal.
- PostgreSQL menerima tambahan queue workload yang harus diindex dan dimonitor.
- Worker dapat dipindah antara long-running container dan scheduled execution.
- Duplicate delivery tetap mungkin dan wajib ditangani idempotently.
