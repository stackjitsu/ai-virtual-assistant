# syntax=docker/dockerfile:1
FROM golang:1.26.3-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY database/ ./database/
COPY web/ ./web/
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/assistant ./cmd/assistant

FROM debian:bookworm-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build --chmod=0555 /out/assistant /app/assistant
USER 65532:65532
# Render supplies PORT. Reject missing/invalid values rather than binding loopback.
# The shell expands only quoted environment values; exec preserves signal handling.
CMD ["/bin/sh", "-c", "set -eu; : \"${PORT:?PORT must be set}\" \"${ASSISTANT_ORIGIN:?ASSISTANT_ORIGIN must be set}\" \"${WIDGET_PARENT_ORIGINS:?WIDGET_PARENT_ORIGINS must be set}\"; case \"$PORT\" in *[!0-9]*|'') echo 'invalid PORT' >&2; exit 1;; esac; [ \"$PORT\" -ge 1 ] && [ \"$PORT\" -le 65535 ] || exit 1; export LISTEN_ADDR=\"0.0.0.0:$PORT\"; exec /app/assistant serve"]
