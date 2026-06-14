## 1. Schema & DB access

- [x] 1.1 Add migration `0017_jobs_moderation.sql`: `users.role` (TEXT NOT NULL DEFAULT 'user', CHECK in user/moderator/admin), `jobs.created_by` + `jobs.updated_by` (BIGINT REFERENCES users(id), nullable)
- [x] 1.2 Add a slim `GetUserRole :one` query in `queries/users.sql` (role by id) for the hot middleware path — leaves the existing `GetUserByID` row shape untouched
- [x] 1.3 Add `UpsertManualJob :one` to `queries/jobs.sql`: fixed `source='manual'`, writes `created_by` on INSERT and `updated_by` on `ON CONFLICT (source, external_id) DO UPDATE`; enqueue stays a separate call
- [x] 1.4 Add `UpdateManualJob :one` to `queries/jobs.sql`: partial update by `WHERE public_slug = $1 AND source = 'manual'` using `COALESCE($field, jobs.field)`, set `updated_by` + `updated_at = now()`, RETURNING the row
- [x] 1.5 Run `make sqlc` and commit the regenerated `internal/db`

## 2. Shared derivation helper

- [x] 2.1 Factor the geo/skills/slug/work-mode derivation out of `pipeline.normalizeJob` into a reusable helper (RED: test the helper's output for a sample input)
- [x] 2.2 Re-point `pipeline.normalizeJob` at the helper; confirm existing pipeline tests stay green

## 3. Role authorization (`internal/auth`)

- [ ] 3.1 Add `RequireRole(q, role)` middleware: reads `userID` from Locals, loads role via `GetUserByID`, 403 on mismatch, 401 when unauthenticated (RED: middleware test for pass / 403 / 401)
- [ ] 3.2 Define the `RoleLoader` interface (or reuse the queries dependency) so the middleware is unit-testable with a fake

## 4. Moderation service (`internal/moderation`)

- [ ] 4.1 Define `Service` + `Repository` interface mirroring `internal/jobtracking`; `Repository` adapts `*db.Queries` + pool for the transactional create/update
- [ ] 4.2 Implement `Create(ctx, actorID, input)`: validate (url/title/company required, url is http(s)) → derive via the helper → tx: `UpsertManualJob` + `EnqueueJobEnrichment` (RED: validation rejects missing/invalid fields; success derives + returns job)
- [ ] 4.3 Implement `Update(ctx, actorID, slug, patch)`: partial update via `UpdateManualJob`, nil field = unchanged, identity untouched (RED: partial update changes only supplied fields; non-manual/unknown slug → not-found error)

## 5. Handler & routes (`internal/handler`)

- [ ] 5.1 Add `jobs_moderation.go`: `CreateJob` (201) and `UpdateJob` (200) handlers — parse body, call `moderation.Service`, return `{ "data": job }`; errors flow through the central `ErrorHandler`
- [ ] 5.2 Wire routes in `Register`: `api.Post("/jobs", RequireAuthOrKey, RequireRole("moderator"), h.CreateJob)` and `api.Patch("/jobs/:slug", RequireAuthOrKey, RequireRole("moderator"), h.UpdateJob)`; construct the `moderation.Service` once in `Register`
- [ ] 5.3 Confirm `created_by`/`updated_by` are absent from the `jobview` wire shape (no change needed; assert in a handler test)

## 6. Integration tests (`-tags=integration`, testcontainers)

- [ ] 6.1 `UpsertManualJob`: idempotent on URL (re-POST updates, no duplicate), sets `created_by` on insert and `updated_by` on conflict, enqueues an outbox row
- [ ] 6.2 `UpdateManualJob`: updates a manual job; leaves a non-manual job untouched (returns no row → 404 path)

## 7. Verify & finish

- [ ] 7.1 `go build ./... && go vet ./... && go test ./...` green; integration suite green
- [ ] 7.2 Manual smoke: grant a moderator via psql, `POST`/`PATCH` with an API key, confirm 201/200, 403 for a non-moderator, 401 unauthenticated
- [ ] 7.3 Note follow-up: `freehire jobs add` / `jobs edit` CLI command in the separate `freehire-cli` repo (out of scope for this change)
