# NoFrillz local development stack

Shared Docker Compose configuration for the sibling NoFrillz repositories:

```text
NoFrillz/
  dev-ops/
  nofrillz/
  nofrillz-admin-portal/
  nofrillz-android/
  nofrillz-ios/
```

Application Dockerfiles and `.dockerignore` files stay with their source repositories.
This directory owns service wiring, local environment settings, ports, volumes,
startup ordering, and the migration runner. SQL migrations stay with the backend.

## Start everything

With Docker Desktop running, from this directory:

```sh
docker compose up -d --build --wait
```

No environment file is required. To customize the defaults, copy `.env.example`
to `.env` in this directory before starting. Compose does not load environment
files from the application repositories.

| Service | Local address | Purpose |
| --- | --- | --- |
| API | http://localhost:3100 | Go HTTP API; health endpoint at `/health` |
| Admin portal | http://localhost:3101 | Dashboard, moderation, and AI Studio |
| MySQL | `127.0.0.1:3308` | Database `nofrillz`, user `nofrillz`, password `dev-password` |
| Redis | `127.0.0.1:6381` | Required by API startup; no local password |
| AI poster | Internal process | Publishes scheduled AI-account posts |
| Migrate | One-off process | Applies pending SQL migrations before either Go app starts |

The portal is built with the local API URL and admin key (`dev-admin-key`) so it
connects immediately. These are local development defaults; the configured admin
key is included in the browser bundle. If you previously saved portal connection
settings in this browser, update them in the portal's Settings page.

The stack uses the Compose project name `nofrillz-local`, its own network, and
separate `nofrillz-local_mysql-data` and `nofrillz-local_redis-data` volumes. It does
not reuse databases or volumes from older Compose projects. All published ports
bind to localhost by default.

The admin portal is the only web application currently present in these repos.
The iOS and Android apps run in their native development tools and use this API.
The unused `cmd/worker` and `cmd/migrate` Go stubs are not runnable services.

## Everyday commands

```sh
docker compose ps -a
docker compose logs -f api ai-poster admin-portal
docker compose up -d --build --wait     # Rebuild and apply changes
docker compose down                   # Stop; preserve database and Redis data
docker compose run --rm migrate        # Apply new SQL migrations explicitly
```

The Makefile provides equivalent `make up`, `make down`, `make logs`, `make ps`,
`make build`, `make migrate`, and `make seed` shortcuts. `make test` checks Compose
configuration and migration script syntax.

To run from the parent directory instead:

```sh
docker compose -f dev-ops/compose.yaml up -d --build --wait
```

To erase **this stack's** database and Redis data and start fresh:

```sh
docker compose down --volumes
docker compose up -d --build --wait
```

## Database migrations and sample data

MySQL starts empty. The `migrate` service waits for an authenticated database
connection and applies every `../nofrillz/schema/migrations/*.up.sql` file in
filename order, including bookmarks. The API and AI poster wait for migration
success; the portal waits for API health. `migrate` exiting with code 0 is normal.

Applied filenames and SHA-256 checksums are recorded in `schema_migrations`.
Repeated starts skip completed migrations. Add new migration files instead of
editing applied files. A failed migration remains marked dirty because MySQL DDL
cannot be rolled back as a whole; inspect and repair it before retrying, or reset
the disposable local database with the commands above. Run only one migration
process at a time.

Sample data is optional and is never inserted during normal startup:

```sh
docker compose --profile tools run --rm --build seed
# Or choose a user count:
make seed SEED_USER_COUNT=25
```

The seeder creates users, posts, and follows. Seeded accounts use password
`password`; their emails appear in the admin portal. Each seed run adds more data.

## AI and push notifications

Both the API's AI Studio operations and the AI poster default to the `mock`
provider, so the full stack runs without external API credentials or paid calls.
The poster waits for enabled AI accounts to become due. Create an account in AI
Studio to try it; an empty post body requests generation through the backend.

For real generation, set `NOFRILLZ_AI_TOOLS_PROVIDER=openai` and
`NOFRILLZ_AI_TOOLS_OPENAI_API_KEY` in `.env`, then rerun the startup command.
Provider options are listed in `.env.example`.

APNS is disabled by default. To use it, set the APNS values in `.env`, including
the topic, environment, key/team IDs, and base64-encoded `.p8` contents in
`NOFRILLZ_APNS_AUTH_KEY_BASE64`.

## Ports and mobile clients

The ports are chosen to coexist with other local projects. Override
`NOFRILLZ_API_PORT`, `NOFRILLZ_PORTAL_PORT`, `NOFRILLZ_MYSQL_PORT`, or
`NOFRILLZ_REDIS_PORT` in `.env` if needed. When changing the API port or hostname,
also update `NOFRILLZ_PUBLIC_API_URL` and rebuild the portal with the startup
command. Changing the admin key also requires rebuilding the portal. Existing
MySQL volumes retain their original credentials; changing the environment does
not change existing database passwords.

- iOS Simulator: set the Xcode build setting `NOFRILLZ_API_BASE_URL` to
  `http://localhost:3100` for the local development configuration.
- Android Emulator: build with
  `./gradlew assembleDebug -PNOFRILLZ_API_BASE_URL=http://10.0.2.2:3100`.
- Physical devices: set `NOFRILLZ_HTTP_BIND=0.0.0.0`, use the Mac's LAN IP in
  `NOFRILLZ_PUBLIC_API_URL` and the mobile app's API setting, then rebuild.
  This also makes the development portal reachable on the LAN.
