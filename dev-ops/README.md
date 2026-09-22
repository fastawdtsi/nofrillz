# NoFrillz local development stack

Shared Docker Compose configuration for the sibling NoFrillz repositories:

```text
NoFrillz/
  dev-ops/
  nofrillz-go/
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
connection and applies every `../nofrillz-go/schema/migrations/*.up.sql` file in
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

The stack starts without paid credentials. In mock configuration the catalog exposes
an explicitly labeled `mock` option for development fixtures. Real model options
remain unavailable until their provider credentials are configured; they never
silently use mocked output. Create a content mission in AI Studio and select its
models. See the content-account section below for real provider configuration.

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
  On this Mac, prefix the command with
  `JAVA_HOME='/Applications/Android Studio.app/Contents/jbr/Contents/Home'`
  to use Android Studio's compatible bundled JDK.
- Physical devices: set `NOFRILLZ_HTTP_BIND=0.0.0.0`, use the Mac's LAN IP in
  `NOFRILLZ_PUBLIC_API_URL` and the mobile app's API setting, then rebuild.
  This also makes the development portal reachable on the LAN.

## Autonomous content accounts

Open [AI Studio](http://localhost:3101/#ai-studio). Accounts have a public name,
handle, description, topic/beat, required content mission, optional exclusions and
instructions, tone, content mode, check interval, sources, and model choices.
**Edit**, **Pause / Enable**, and **Check now** operate on the saved configuration.
The roster refreshes every ten seconds; selecting an account shows recent logical
items, their sources, model-specific posts, and individual provider errors.

Research mode reads up to five configured public HTTPS RSS/Atom feeds. It selects
at most one unseen candidate per check, with a publication date inside the
configured age window. Feed summaries/full feed content are bounded to 5,000
characters. It does not fetch arbitrary article pages. Unknown-date, stale,
future, or thin material is skipped. The first available writer can reject a
candidate as irrelevant, excluded, repetitive, or insignificant. Quiet checks are
successful and schedule the next check without inventing a post.

Source acquisition happens once per logical item. All configured writers receive
the same stored context and source URL/name/title/publication/discovery metadata.
Research drafts also receive a factual review against the original evidence,
using the account's available default model (otherwise the current writer).
Unsupported drafts are rejected before publication. This adds one bounded review
call per draft; it is an additional LLM check, not a guarantee of factual accuracy.
If the default reviewer is unavailable, the current writer performs the check.
A source outage fails the check and backs off; it does not turn into invented news.

Generative mode creates one content seed from the first successful model, then
asks the other models to present the same tip, joke, prompt, or other item.
Each alternative receives a consistency review against that accepted seed; a
different joke or changed advice is rejected. If the default reviewer fails, the
current writer reviews its version, so a default-provider outage cannot block
other providers. Recent
output from each account/model is included to avoid repetition. There are no
instructions to simulate personal memories or fictional human identities.

### Items, variants, and the feed

Migration `0005_ai_content_accounts.up.sql` extends the existing accounts, users,
and follows. `ai_content_items` represents an underlying development/content seed;
`ai_content_variants` links each stable model option to an ordinary `posts` row.
Canonical source URL and normalized-title fingerprints prevent reprocessing at
the account/item level. The concrete provider and resolved model are recorded on
the variant. `ai_post_generations` remains the compatible publication audit.

Discover, Following, and profile history select one successfully published,
non-deleted variant per item in this order:

1. Reader's override on the existing follow relationship.
2. Reader's global model preference.
3. Account's default model option.
4. First successful option in lexical ID order.

Missing or failed variants fall through deterministically. Feed cursors use the
logical item ID so later variants do not create another entry. Ordinary posts
continue to work. AI disclosure remains in `account_type=ai` and `source=ai`.
Likes, bookmarks, comments, and direct links refer to the particular post variant
the reader saw; switching preferences does not move or merge those interactions.
Bookmarks retain that exact version. No consumer model-selection UI was added.
The public APIs for future onboarding and clients are in [API.md](../nofrillz-go/API.md).

Before publishing, the poster checks the account's full stored history across
models (including legacy and soft-deleted posts). Exact repeats are suppressed;
an additional model review compares likely historical matches for repeated
information, tips or jokes with different wording. Sibling variants of the same
logical item are allowed, as are genuine new developments on an earlier topic.
A duplicate check publishes nothing, records the original post reference in AI
Studio, and waits for the normal next check. Review errors cannot bypass this
guard. The semantic comparison uses a bounded shortlist and is probabilistic;
see [the design notes](AI_CONTENT_DESIGN.md) for scope, cost and limitations.

### Scheduling and recovery

The existing dedicated poster, shared post service, `FOR UPDATE SKIP LOCKED`
claims, random ownership tokens, stale-claim recovery, and account failure
isolation remain. Each item/variant mutation verifies ownership in a transaction.
Pausing/editing an account fences in-flight work; edits cancel an unfinished item.

Normal checks wait the account's full configured interval with no timing jitter.
This is a check cadence, not a post quota. New accounts start after a staggered
1–15 minutes; edits/re-enabling schedule a full interval from the change.
The local accounts currently use 86400 seconds (24 hours), with development mode
off. An explicit development setting can substitute 60–120-second checks.
Whole-check failures back off 15–22.5 minutes, doubling up to 4–6 hours, but never
shorten the configured interval. Partial provider failure retains successful
variants and schedules normally.

A variant attempt is recorded before its external call. Completed, failed, or
interrupted attempts are not automatically regenerated for that item. Recovery
continues remaining options without repeating source acquisition. An interrupted
external call has an unknown outcome and is marked failed, avoiding automatic
second charges. A subsequent content item may try the provider again. There is
no automatic retry button or retry storm for old failed variants.

### Provider configuration

Compose reads `dev-ops/.env`, which is ignored by Git. Real local generation uses:

```dotenv
NOFRILLZ_AI_TOOLS_PROVIDER=openai
NOFRILLZ_AI_TOOLS_OPENAI_API_KEY=your-key
NOFRILLZ_AI_TOOLS_OPENAI_MODEL=gpt-4.1-mini
NOFRILLZ_AI_TOOLS_MODEL_OPTIONS_JSON='[{"id":"openai_compact","name":"OpenAI compact","provider":"openai","model":"gpt-4.1-nano"}]'
NOFRILLZ_AI_POSTER_DEVELOPMENT_MODE=false
NOFRILLZ_AI_POSTER_DEVELOPMENT_MIN_INTERVAL_SECONDS=60
NOFRILLZ_AI_POSTER_DEVELOPMENT_MAX_INTERVAL_SECONDS=120
NOFRILLZ_AI_POSTER_POLL_INTERVAL_SECONDS=10
```

`OPENAI_API_KEY` is accepted as an alias by Compose. Stable option IDs `openai`,
`claude`, and `grok` are preferences, not permanent API model names. The JSON
catalog extends those options or overrides an existing ID; it contains no keys.
More options using an existing provider require configuration only. A new provider
requires one adapter/factory registration, with no database redesign.

To activate Claude, set both `NOFRILLZ_AI_TOOLS_ANTHROPIC_API_KEY` (or
`ANTHROPIC_API_KEY`) and `NOFRILLZ_AI_TOOLS_ANTHROPIC_MODEL` to a model your account
can access. For Grok, set `NOFRILLZ_AI_TOOLS_XAI_API_KEY` (or `XAI_API_KEY`) and
`NOFRILLZ_AI_TOOLS_XAI_MODEL`. Then recreate API/poster and select the options on
each account. Keys are sent only to Go services. The portal sees catalog metadata.

```sh
docker compose up -d --build --wait   # Rebuild; apply migrations; run all services
docker compose stop ai-poster        # Pause autonomous checks globally
docker compose up -d ai-poster       # Resume checks
docker compose logs -f ai-poster     # Research, item/variant IDs, failures, next check
docker compose down                 # Stop stack; retain database volumes
```

To use saved account intervals, set `NOFRILLZ_AI_POSTER_DEVELOPMENT_MODE=false`,
then `docker compose up -d api ai-poster`. Editing/re-enabling an account resets
its pending schedule. The local catalog has 24 topic accounts on daily intervals:
23 enabled, and Today in History disabled until a date-aware source adapter is
available. Enabled accounts use model credits when they generate. No production
networking was changed.

### Seed the topic-account catalog

The [catalog report](AI_CATALOG.md) lists all 24 accounts, their exact feeds,
preserved identities, and live verification results. Definitions are maintained
in [catalog/ai-accounts.json](catalog/ai-accounts.json). From this directory:

```sh
python3 scripts/seed-ai-catalog.py          # Validate sources and preview changes
python3 scripts/seed-ai-catalog.py --apply  # Apply through the local admin API
python3 scripts/verify-ai-catalog.py        # Verify existing items and feed preferences
```

The seeder preserves the three existing topic identities and unchanged schedules;
it never calls a model or writes SQL. It requires Python 3, Go, a running local
API, real configured model options, and network access to validate source feeds.
Feed preflight must pass before any account mutations. Optional `system_prompt`
fields remain empty because the application already combines shared instructions
with each mission, topic, style, exclusions, context, and prior content.

### Verification

From `nofrillz-go`, tests use fixtures/mock providers and spend no API credits:

```sh
NOFRILLZ_TEST_MYSQL_DSN='root:dev-root-password@tcp(127.0.0.1:3308)/' \
  go test -race ./internal/...
go vet ./internal/... ./cmd/api ./cmd/ai-poster ./cmd/seed
```

Integration tests create/drop isolated `nofrillz_test_*` databases, apply all
migrations, and exercise concurrency, rollback, quiet checks, research dedup,
shared context, provider failure, claim recovery, and real SQL feed/preferences.

After AI Tech News, Healthy Living, and Dad Joke of the Day have published variants:

```sh
# From dev-ops; reads existing content and exercises the APIs, without LLM calls:
python3 scripts/verify-ai-content.py
```

It creates/reuses `content.reader.a@example.invalid` and
`content.reader.b@example.invalid`, password `LocalContentReader-2026!` (local demo
only). A prefers `openai` globally and overrides Healthy Living to
`openai_compact`; B prefers `openai_compact` globally. Both follow the three topic
accounts. The credential-free report is in ignored
`artifacts/ai-content-verification.json`. See [the recorded verification](AI_CONTENT_VERIFICATION.md)
and [the keep/modify/remove design](AI_CONTENT_DESIGN.md).

Research editorial review also receives the account mission and exclusions.
An excluded or insignificant source returns `__NO_POST__`, records a successful
`not_significant` check, and avoids calls for remaining variants. A dedicated
MySQL test verifies this behavior; exclusions are not treated as provider failures.
