# STAGE-02 foundation handoff — stop after local checkpoint

Date: 2026-11-05 UTC. Owner: scoped project-foundation deployment-preparation and primary-source research worker. Branch: `release/visitor-staging`. Source baseline: **8410640e027c732a83dc35d354f8c4a56736be0e**. Preparation delivered; **container acceptance BLOCKED**, not deployed/live/release-ready. No recursive delegation.

## Delivered and reviewed

- `Dockerfile`: Go 1.26.3 multi-stage build of the existing monolith; no frontend build. CGO disabled, trimmed paths/VCS metadata, stripped binary. Nonroot Debian runtime with CA certificates; only the compiled application copied from build stage. No source, module cache, credentials, migrations execution or seed execution copied/run as runtime startup tasks. Embedded public widget/SQL assets are in the binary as intended.
- `.dockerignore`: deny-default context with explicit known source/SQL/public-asset allowlist; test files excluded. History, secrets/env files, databases, scratch and node caches are not admitted. Source/asset credential review still matters: filtering cannot detect secrets disguised as source. Actual engine context inspection is NOT RUN.
- `render.yaml`: one Docker web service, explicit `plan: free`, Git automatic deploy trigger off, no disk/database/addon/hooks. Existing `/widget/widget.css` static health path; does not certify remote schema, content or inference. `STORE_MODE=turso`, agreed demo flag and Gemini endpoint/model. Operator-entered exact origins and provider/Turso secrets use `sync: false`, never generated provider keys or committed real identifiers. No Blueprint was created/synced; first creation can deploy despite automatic Git deploys being off.
- `docs/deployment/render-staging.md`: explicit future human/account steps, no-card/free-only stop policy, strict actual HTTPS origins, separate schema migration and reviewed content import, fictional/not-official limited pilot, privacy/retention/spend limitations, no-transcript/key logging, cold starts, accepted-commit rollback and authorized teardown. Placeholder `example.org` sources remain truthfully non-resolving citations; no remote corpus claim.
- `.env.example`, `README.md`: additive staging guidance. No auto-load `.env` claim; baseline image explicitly must not be published before separate demo/email-disable policy integration.
- `.scratch/staging/provider-notes.md`: one primary-source findings document, URLs/retrieval date, eligibility unknowns and negative fetch results. Public HTML/text evidence remains task-local under `sources/` (not included in the checkpoint); response hashes and initial retrieval metadata are included.

**No application/runtime/web/contracts/dependencies/workflows/central-ledger changes.** `git diff HEAD --exit-code -- cmd internal database web docs/contracts go.mod go.sum .github .harness Makefile scripts` returned 0. No secret files were read, environment dumped, accounts inspected, global tools installed, model requests made, remote data changed, resources provisioned, services started/deployed, branches merged or pushes performed. Only read-only public documentation GETs and ordinary bounded local build/test tooling were used.

## Exact evidence / hashes / diff

- Source input list: `checks/context-source-paths.txt` (25 tracked build-source inputs at baseline).
- Per-file source SHA-256: `checks/source-sha256.txt`; manifest SHA-256 **b69648ea4fbec02d0afb78f82506a192f6e03e7c3df9f8088a85d256b1df84ca**.
- Exact 7 deployment/research deliverable hashes: `checks/deliverable-sha256.txt`; manifest SHA-256 **5c6dc879df6df77dceda29d1c90c258b5459118ca2c5ba672dba0e4ccb40f9e7**.
- Downloaded primary-source/evidence hashes: `checks/provider-sha256.txt`; native build artifact hash: `checks/artifact-sha256.txt`.
- `implementation.patch` is the full staged implementation/evidence diff **before this handoff** (task-local, not included in commit); SHA-256 **c4a06c2cbbdf2f517909d7f0dfc6c1ea2081dcc41bc2cbabd7df8bac648f5b9d**. The checkpoint's full authoritative diff including this handoff is reproducible with `git show --format=fuller --stat HEAD` / `git diff 8410640e027c732a83dc35d354f8c4a56736be0e HEAD` after checkpointing.
- Check logs: `checks/{format,lint,static,unit,contract,schema,focused,cross-build}.log` are local evidence, ignored by Git. `checks/results.json`, focused check script and hash manifests are included in the checkpoint. Compiled binary and fetched HTML/text are **not** committed.

## Bounded commands and actual outcomes

All check subprocesses explicitly had `OPENAI_API_KEY=`, `TURSO_AUTH_TOKEN=`, `TURSO_DATABASE_URL=`. Existing read-only check venv used by putting `/home/me/Documents/agent-worktrees/assistant-visitor-rc/.scratch/release/checks-venv/bin` first on PATH; no changes to that tree. Runner also clears credentials for child tests. No exported credentials were printed/read for this work.

| Command | Bound / result |
|---|---|
| `make format` | Via `timeout --signal=TERM --kill-after=5s 300s`; PASS, 0.22 s. |
| `make lint` | Same 300 s bound; PASS, 1.37 s. |
| `make static` | Same 300 s bound; PASS, 2.53 s; real schema checks and compile-only Go tests. |
| `make unit` | Same 300 s bound; PASS, 35.39 s; Python tooling tests, Go race suites, 66 Node tests passed. |
| `make contract` | Same 300 s bound; PASS, 13.45 s; Go contract/race suites and 15 Node contract tests passed. Precautionary run for deployment/config guidance; no contract file edited. |
| `python3` + `yaml.safe_load(render.yaml)` | 20 s bound; PASS with system PyYAML; writes parsed JSON only in task scratch. |
| Existing check venv Python + `jsonschema.Draft202012Validator` | 20 s bound; PASS against downloaded official Render schema, offline/local; no Render validation API request. |
| `python3 .scratch/staging/checks/focused.py` | 30 s outer bound; PASS. `/bin/sh -n` and mock-exec tests each bounded to 5 s; valid numeric ports map to `0.0.0.0:$PORT`, invalid/injection-shaped/missing ports and missing origins fail. One free service, nine expected env entries, non-generated secrets, no hooks/paid resources, nonroot/CA structure and context filename allowlist reviewed. This is NOT a Dockerfile parser, native context calculation or actual app/container run. |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags='-s -w' -o .scratch/staging/checks/assistant-linux-amd64 ./cmd/assistant` | 180 s + 5 s kill grace; PASS. `go version`: go1.26.3 linux/amd64. `file` reports stripped, statically linked Linux x86-64 ELF. Native Linux cross-build-mode evidence only, NOT Docker build/runtime evidence. Binary not executed. |
| `git diff --check`, `git diff --cached --check` | 10 s each; PASS. Final path-boundary review found only allowed paths. |
| Docker/Podman build; image/layer inspection; container boot, nonroot/HTTPS/port smoke | **NOT RUN — BLOCKER: neither `docker` nor `podman` is installed/available on PATH.** No installation/daemon provisioning attempted. `hadolint`/`shellcheck` also unavailable; no claims for these tools. |
| Native fuzz-smoke/full, combined browser acceptance, live AI/Turso, hosted CI | NOT RUN in this task. No application parser/state transition was changed. Coordinator owns integration/full fuzz and combined-browser gates; remote/live actions remain separately unauthorized. Existing unit suites include fuzz seed execution, not a new native fuzz campaign. |

One bounded preparation repair: quote YAML `"off"` so YAML 1.1 parsers do not coerce it to boolean. No failed check required a repair. No scope expansion.

## Current primary-source references (retrieved 2026-11-05)

- https://render.com/docs/blueprint-spec and https://render.com/schema/render.yaml.json — Docker/free/auto-deploy/health fields, `sync:false` initial-only prompts and official structure schema.
- https://render.com/docs/web-services — `0.0.0.0` + `PORT` binding and public/private accessible repository support.
- https://render.com/docs/free and https://render.com/docs/faq — sleep, ephemeral storage, quotas/SMTP limits and possible supplementary billing with payment method. **Actual card/private-repo/free-Blueprint account eligibility remains unverified.**
- https://render.com/docs/infrastructure-as-code — creation/provisioning, separate Auto Sync setting and deletion/recreation behavior.
- https://render.com/docs/configure-environment-variables and https://render.com/docs/github — runtime configuration / Git-provider access.
- https://turso.tech/pricing — advertised no-card free plan/quotas, overages once enabled; no actual account entitlement checked.
- https://docs.turso.tech/sdk/http/reference and https://docs.turso.tech/cli/db/shell — HTTPS/bearer pipeline and explicit SQL operations; no remote queries/migrations performed.

## Blockers and next owner decisions

1. **Coordinator/container-capable validator:** perform actual Docker/Podman build, inspect real context/final layers, test nonroot runtime, CA certificates and `0.0.0.0:$PORT` binding with fixture-only/no-provider credentials. Base-image availability, native layer safety, Render image behavior and actual container health are unverified here. Do not substitute cross-build success for this blocker.
2. **Coordinator + backend/widget owner:** integrate and accept separate `ASSISTANT_PUBLIC_DEMO=true` demo-label/email-disable runtime policy; rebuild and test the exact combined commit. This checkpoint packages baseline source, not that later policy. Runtime policy absent on baseline is an explicit public-exposure blocker.
3. **Coordinator/human operator:** resolve actual provider eligibility, no-billing controls, privacy/retention/deletion notice, exact future hostname/origins, remote migration/content-import execution sequence, reviewed resolving source content, and distinct authorization before any account/resource/data/model operation. Stop if card/payment/paid upgrade/overages are required. No present setup authority exists.
4. **Coordinator:** combined independent validation and full fuzz before release; no production claim from this checkpoint. No live AI check is claimed (explicitly skipped by user direction).

Local checkpoint contains only preparation and bounded local evidence; it is not release acceptance. Identify its hash using `git log -1 --format=%H` in this worktree after checkpoint creation (reported to coordinator in the final response). No merge/push/main/hosted action is authorized. **Worker stops after recording this handoff/checkpoint.**
