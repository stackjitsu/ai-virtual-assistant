#!/usr/bin/env bash
set -euo pipefail
cd /home/me/Documents/formartting/Working_Milestone_Files/ai-virtual-assistant
EVIDENCE="$PWD/.scratch/widget-ux/evidence/ux-02"
exec env -i PATH="$PATH" HOME="$EVIDENCE/preview" STORE_MODE=sqlite \
  SQLITE_PATH="$EVIDENCE/preview/assistant.sqlite" OPENAI_API_KEY= TURSO_DATABASE_URL= TURSO_AUTH_TOKEN= \
  ASSISTANT_ORIGIN=http://127.0.0.1:18102 LISTEN_ADDR=127.0.0.1:18102 \
  WIDGET_PARENT_ORIGINS=http://127.0.0.1:18102 \
  "$EVIDENCE/assistant" serve --seed-demo >> "$EVIDENCE/preview/backend.log" 2>&1
