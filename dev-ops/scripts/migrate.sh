#!/bin/sh
set -eu

query() {
  mysql --protocol=TCP --host="$MYSQL_HOST" --user="$MYSQL_USER" \
    --database="$MYSQL_DATABASE" --batch --skip-column-names "$@"
}

query <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  filename VARCHAR(191) NOT NULL PRIMARY KEY,
  checksum CHAR(64) NOT NULL,
  dirty BOOLEAN NOT NULL DEFAULT TRUE,
  applied_at TIMESTAMP(6) NULL
) ENGINE=InnoDB;
SQL

for migration in /migrations/*.up.sql; do
  [ -f "$migration" ] || { echo "No migrations found in /migrations" >&2; exit 1; }
  filename=$(basename "$migration")
  case "$filename" in
    *[!a-zA-Z0-9_.-]*) echo "Invalid migration filename: $filename" >&2; exit 1 ;;
  esac
  checksum=$(sha256sum "$migration" | cut -d ' ' -f 1)
  previous=$(query -e "SELECT checksum FROM schema_migrations WHERE filename='$filename'")

  if [ -n "$previous" ]; then
    if [ "$previous" != "$checksum" ]; then
      echo "Migration changed after being recorded: $filename. Add a new migration instead." >&2
      exit 1
    fi
    dirty=$(query -e "SELECT dirty FROM schema_migrations WHERE filename='$filename'")
    if [ "$dirty" != "0" ]; then
      echo "Migration $filename did not finish. Inspect and repair the database before retrying." >&2
      exit 1
    fi
    echo "Already applied: $filename"
    continue
  fi

  echo "Applying: $filename"
  # MySQL DDL commits implicitly. Keep a dirty record if a migration fails midway.
  query -e "INSERT INTO schema_migrations (filename, checksum) VALUES ('$filename', '$checksum')"
  query < "$migration"
  query -e "UPDATE schema_migrations SET dirty=FALSE, applied_at=CURRENT_TIMESTAMP(6) WHERE filename='$filename'"
done

echo "Database migrations complete."
