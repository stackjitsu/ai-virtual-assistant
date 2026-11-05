# Combined staging preparation — coordinator checkpoint

Candidate preparation combines foundation commit 7132468 and policy commit 8754b50.
Both owners independently used .scratch/staging/handoff.md, creating an add/add evidence-only conflict. Coordinator preserved BOTH original handoffs verbatim as foundation-handoff.md and policy-handoff.md beside this index; no application/source conflict required resolution. No source was replaced with ours/theirs.

- foundation-handoff.md: Render/Docker preparation, official provider research, static/schema/shell/Go checks, missing local container engine blocker.
- policy-handoff.md: contract-first public demo policy, disabled UI/server email collection, owner gates/full fuzz/default+demo trusted browser evidence. Its original uncommitted wording describes owner delivery; coordinator subsequently verified all 15 source hashes and checkpointed the exact tested sources as 8754b50 without implementation edits.

This local merge is NOT independent combined-tree acceptance, a container build, hosted CI, live-model evaluation or deployment. User explicitly skipped live AI testing. Independent validation follows on an immutable detached worktree; no remote resources, pushes, secret uploads, migrations/imports or billing changes authorized/performed.
