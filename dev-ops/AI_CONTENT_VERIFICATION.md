# Content-account verification — September 22, 2026

The corrected system is running locally. All three accounts were created through AI Studio. The public API demonstration uses real stored posts; automated tests use fixtures and never fabricate live provider success.

## Accounts and configuration

| Account | Mode / saved interval | Mission |
| --- | --- | --- |
| AI Tech News (`@ai_tech`) | research / 30 minutes | Find and summarize meaningful developments in AI models, research, products, policy, and infrastructure. Use only facts in the supplied sources. Attribute announcements to the source and include the publication date when useful. Skip material that is merely promotional, repetitive, or too thin to support a useful update. |
| Healthy Living (`@healthy_living`) | generative / 1440 minutes | Offer one useful, practical, low-risk everyday wellbeing tip. Rotate among gentle movement, regular sleep routines, taking breaks, and making healthy habits easier. Give general information rather than personalized medical advice. Avoid repeating recent tips. |
| Dad Joke of the Day (`@dad_joke_day`) | generative / 1440 minutes | Generate one short, clean dad joke. Keep the setup and punchline concise. Avoid repeating recent joke premises or punchlines. This is an openly AI-powered humor account, not a fictional parent. |

All three are enabled with `openai` → `gpt-4.1-mini` and `openai_compact` → `gpt-4.1-nano`, account default `openai`. Development scheduling substitutes randomized 60–120-second checks, with a 10-second poll. AI Tech News uses `https://openai.com/news/rss.xml`, a seven-day freshness window, explicit promotional/unsupported-claim exclusions, and concise source-attributed summaries. The two evergreen accounts do not claim current external research.

## Actual database relationships

`users.id → ai_accounts.user_id → ai_content_items.ai_account_id → ai_content_variants.content_item_id → posts.id` (via the variant’s `post_id`).

| Account ID | Logical item ID | OpenAI post | Compact-model post |
| --- | --- | --- | --- |
| `162345322113138689` (ai_tech) | `162356638538269696` | `162356671497110528` | Rejected; fallback uses the OpenAI post |
| `162345451641634816` (healthy_living) | `162356691663324160` | `162356700714632192` | `162356715738629120` |
| `162345576480899073` (dad_joke_day) | `162357057968669696` | `162357067003200512` | `162357085021930496` |

Both live choices use OpenAI. The successful calls returned concrete versioned models `gpt-4.1-mini-2025-04-14` and `gpt-4.1-nano-2025-04-14`. Claude and Grok adapters passed mocked HTTP contract and partial-failure tests; their real APIs were not called because credentials are absent.

## Real research and quiet checks

The recorded research item uses [Introducing the Australian Youth Safety Blueprint](https://openai.com/index/australian-youth-safety-blueprint), published `2026-09-18T12:00:00Z` and discovered `2026-09-22T16:49:29.534190513Z`. The stored source context describes OpenAI's six-pillar Australian Youth Safety Blueprint. The published version is:

> On September 18, 2026, OpenAI introduced the Australian Youth Safety Blueprint, a six-pillar framework designed to create safer AI experiences that specifically protect and empower young people (OpenAI News, 2026).

The compact model’s draft failed the source-evidence check. Its failed variant remains recorded without a post ID; the successful OpenAI post remains visible. A reader preferring the compact model receives that successful fallback. Later checks continue; a check where every draft fails validation backs off rather than publishing unsupported material.

A separate real check at `2026-09-22T16:36:55.752536Z` used a one-hour source-age window. It returned `no_content`, kept the item count at 2, preserved the previous successful publication time, and scheduled another check at `2026-09-22T16:38:38.528996Z`. The seven-day window was then restored. No artificial news was supplied.

## Shared generative content

Dad Joke of the Day, item `162357057968669696`:

- OpenAI: Why did the orange stop halfway up the hill? It ran out of juice!
- Compact: Why did the orange stop partway up the hill? It ran out of juice!

Healthy Living item `162356691663324160` likewise has two post IDs under one accepted seed. The full bodies are in the JSON report. Evergreen alternatives are checked against the seed so an unrelated joke or changed advice is rejected.

## Reader preferences and feed proof

Local test logins (password `LocalContentReader-2026!`):

- `content.reader.a@example.invalid`: global `openai`, per-follow override `healthy_living → openai_compact`.
- `content.reader.b@example.invalid`: global `openai_compact`, no account overrides.

Both follow the same three content accounts. The script first clears overrides to demonstrate global selection, then verifies setting/removing an override and leaves A with the Healthy Living override.

| Reader state | Healthy Living logical item | Selected post |
| --- | --- | --- |
| A: global OpenAI, no override | `162356691663324160` | `162356700714632192` |
| B: global compact | `162356691663324160` | `162356715738629120` |
| A: global OpenAI + compact override | `162356691663324160` | `162356715738629120` |

Discover, Following, and profile history each returned exactly one selected variant per item. Setting A’s global preference to unavailable `grok` returned the account’s OpenAI fallback; A’s global preference was restored afterward. IDs remain strings. Likes/bookmarks work on the actual post variant, and the same-reader automated test proves preference changes do not move reactions to another variant.

## Validation completed

- `go test -race ./...` with isolated MySQL integration databases: passed.
- `go vet ./...`: passed.
- Portal `npm run build` and Docker image builds: passed.
- `make test` in dev-ops: Compose configuration and migration-script syntax passed.
- `python3 scripts/verify-ai-content.py`: passed against the running stack.
- Browser verification: create all three accounts, edit research configuration, schedule checks, inspect model availability, sources, variants, failures, and live scheduling.
- Migrations 0001–0005 applied, all `dirty=0`. Repeated starts skip completed migrations.

Automated coverage includes simultaneous claims, stale-worker fencing, pause/edit behavior, write rollback, shared research/context, deduplication, quiet/significance checks, interrupted attempts, partial/default-provider failure, backoff, disabled providers, source freshness/URL restrictions, source and seed validation, authenticated preference APIs, deterministic SQL fallback/pagination, and variant-specific reactions.

## Preserved, replaced, and retired

Preserved: the dedicated poster, scheduler loop, ownership claims, stale recovery, shared posts service, OpenAI transport, admin authentication/account CRUD, Compose, migration history, recent-output checks, and relevant tests.

Replaced: persona/random-thought generation, daily-quota UI, single-model processing, and duplicate-variant feed behavior. Content missions, research acquisition, logical items, configured model variants, reader preferences, and shared selection now drive the product. The obsolete direct AI preview/publish implementation was removed; its endpoints return 410 so they cannot bypass the pipeline.

Retired: the three paused Mara/Dex/Ines demo users and their posts were soft-deleted with the narrowly scoped `scripts/retire-legacy-ai-demo.sql`. Other users and seeded data were preserved. Initial research drafts with unsupported claims and unreviewed compact-model demo variants were removed from the feed, with audits retained. This live inspection prompted the evidence/seed gates and a correction to conflicting “choose a different subject” wording in alternative-variant prompts.

## Running services and commands

| Service | State / address |
| --- | --- |
| API | Running, healthy — http://localhost:3100 |
| Admin portal | Running, healthy — http://localhost:3101/#ai-studio |
| AI poster | Running as a dedicated process |
| MySQL 8.4 | Running, healthy — 127.0.0.1:3308 |
| Redis | Running, healthy — 127.0.0.1:6381 |
| Migrate | Successfully exited (0) |

```sh
cd /Users/ernie/Projects/NoFrillz/dev-ops
docker compose up -d --build --wait
python3 scripts/verify-ai-content.py
docker compose logs -f ai-poster
docker compose stop ai-poster  # stop paid generation; keep the rest running
```

Read-only SQL inspection:

```sh
docker compose exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_PASSWORD" exec mysql -u"$MYSQL_USER" "$MYSQL_DATABASE"' < scripts/inspect-ai-content.sql
```

Credential-free local snapshots are in ignored `artifacts/ai-content-verification.json`, `ai-content-configs.json`, `ai-content-database.txt`, and `ai-quiet-check.json`.

To activate Claude, add `NOFRILLZ_AI_TOOLS_ANTHROPIC_API_KEY` and `NOFRILLZ_AI_TOOLS_ANTHROPIC_MODEL` to `dev-ops/.env`. For Grok, add `NOFRILLZ_AI_TOOLS_XAI_API_KEY` and `NOFRILLZ_AI_TOOLS_XAI_MODEL`. Choose concrete model IDs your provider accounts can access, recreate API/poster, and enable those options on each account. No real keys are in this report or Git.

The initial source adapter reads configured RSS/Atom content, not full arbitrary websites. URL/title fingerprints are practical deduplication, not semantic event clustering. LLM evidence/seed review is an extra quality gate and can make mistakes; it does not guarantee factual accuracy. Comments/bookmarks/likes remain variant-specific. Consumer onboarding/model-choice UI and production deployment were intentionally left for later. The local poster remains accelerated and uses OpenAI credits; no DNS, AWS, UniFi, or public networking changes were made.

Research editorial review also receives the account mission and exclusions.
An excluded or insignificant source returns `__NO_POST__`, records a successful
`not_significant` check, and avoids calls for remaining variants. A dedicated
MySQL test verifies this behavior; exclusions are not treated as provider failures.
