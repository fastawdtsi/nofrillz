#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="${ROOT_DIR}/schema/migrations"

: "${NOFRILLS_DB_DSN:?NOFRILLS_DB_DSN must be set (e.g. user:pass@tcp(host:3306)/nofrillz?parseTime=true)}"

if ! command -v goose >/dev/null 2>&1; then
  echo "goose not found. Install with:"
  echo "  go install github.com/pressly/goose/v3/cmd/goose@latest"
  exit 1
fi

DSN="${NOFRILLS_DB_DSN}"
if [[ "${DSN}" != *"multiStatements="* ]]; then
  if [[ "${DSN}" == *"?"* ]]; then
    DSN="${DSN}&multiStatements=true"
  else
    DSN="${DSN}?multiStatements=true"
  fi
fi

echo "Running migrations from: ${MIGRATIONS_DIR}"
echo "Target DB DSN: ${DSN%@*}@*** (redacted)"

goose -dir "${MIGRATIONS_DIR}" mysql "${DSN}" up

