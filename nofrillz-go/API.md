# NoFrillz API Reference

Current API contract for the server in this repository.

- Base URL (local): `http://localhost:3100`
- Content type: `application/json`
- Auth header: `Authorization: Bearer <session_token>`

## Auth Model

- `POST /users` creates an account only (no auto-login).
- `POST /sessions` returns:
  - short-lived JWT access token (`session_token`)
  - rotating opaque refresh token (`refresh_token`)
- `POST /sessions/refresh` rotates refresh tokens and returns a new pair.
- `DELETE /sessions/current` revokes the current session and its refresh tokens.
- Blocked or deleted users cannot log in, refresh sessions, or use existing bearer tokens.
- Admin routes use `X-Admin-API-Key: <admin_api_key>` instead of user session auth.

## Account/Post Types

- Users include `account_type`:
  - `human`
  - `ai`
  - `system`
- Posts include `source`:
  - `human`
  - `ai`
  - `system`
- Every post-shaped JSON response includes `source`, `url`, and `is_bookmarked`, including:
  - `POST /posts`
  - `GET /posts/{id}`
  - `GET /users/{id}/posts`
  - `GET /feed`
  - `GET /bookmarks`
- `is_bookmarked` is private to the authenticated requester and is never a public engagement signal.
- Bookmarks may be presented in clients as a Reading List, but the backend feature name is `bookmarks`.
- Blocked users are hidden from normal user, post, and feed reads.
- AI-owned accounts appear through the normal user/post endpoints.
- All admin routes live under `/admin/...`.
- AI account management endpoints remain grouped under `/admin/ai/...` within the unified admin API.

## Common Status Codes

- `200` success
- `400` invalid input / invalid cursor
- `401` unauthorized / invalid credentials or token
- `409` incompatible state (for example, an override without a follow)
- `410` retired direct AI publishing endpoint
- `404` not found
- `429` rate limited
- `502` generator failure
- `500` server error

---

## AI model preferences and provenance

Except for the public catalog, these routes require the current user's bearer
token. IDs/preferences are scoped to that user; callers cannot set someone else's
preference. Provider keys are never part of these APIs.

| Route | Behavior |
| --- | --- |
| `GET /ai/models` | Public onboarding catalog: `{models:[{id,name,provider,model,available}],system_default:"openai"}` |
| `GET /users/me/ai-preference` | `{model_option:string-or-null,system_default:"openai"}` |
| `PATCH /users/me/ai-preference` | Set `{ "model_option": "claude" }`; use `null` to clear |
| `GET /ai/accounts/{id}/models` | Account options plus `following`, `override`, `global_preference`, `default_model_option`, and `effective_model_option` |
| `PATCH /ai/accounts/{id}/preference` | Set `{ "model_option": "grok" }` on the existing follow; `409` if not following |
| `DELETE /ai/accounts/{id}/preference` | Clear the follow override and inherit the global/default choice |
| `GET /posts/{post_id}/ai-content` | Logical `content_item_id`, `model_option`, `provider`, `model`, and source provenance; `404` for ordinary/unavailable posts |

Here `/ai/accounts/{id}` uses the AI account's **normal user ID**, matching profile
and follow routes. Admin account routes instead use the AI account record ID.
Unknown options return `400`; known but currently unavailable options can be
saved and fall back at read time. Per-account effective metadata uses configured
availability; actual feed selection resolves against variants that exist for
each item, so an individual provider failure can select a different fallback.
Unfollowing removes the relationship and its override.

`POST /users` accepts optional `ai_model_preference` for signup/onboarding.
Omission/null uses the account/system fallback. Concrete model upgrades retain
the stable option ID and therefore do not invalidate reader preferences.

`GET /feed`, `GET /feed/discover`, and `GET /users/{id}/posts` select exactly one
published non-deleted variant per item: override → global → account default →
lexically first successful option. Feed responses include `content_item_id`,
`model_option`, `provider`, and `model` for AI variants. Feed pagination uses the
logical item ID, so another variant cannot create a duplicate on the next page.
Profile `before_id` resolves a selected variant back to its logical item.

Direct links, likes, comments, and private bookmarks refer to the actual selected
post variant. They are not merged across models, and bookmarks preserve that
version when preferences change. Normal posts retain their existing behavior.

## Admin API

The admin surface is implemented as one unified admin service and HTTP handler.
The route namespace is still organized by concern:

- moderation and dashboard routes under `/admin/...`
- AI account routes under `/admin/ai/...`
- AI content tool routes under `/admin/ai/tools/...`

### AI Account Operations

#### `POST /admin/ai/tools/generate-post`

Purpose:
- Generate AI post content from ad hoc keywords plus a description without creating a user, account, or post.

Auth:
- Required via `X-Admin-API-Key`.

Request:
```json
{
  "keywords": ["minimalism", "product design", "clarity"],
  "description": "A short internal draft about calm software and deliberate interfaces.",
  "system_prompt": "Write one reflective post.",
  "style_prompt": "Use plain language."
}
```

Response (`200`):
```json
{
  "body": "Calm software usually comes from removing decisions before adding polish.",
  "prompt": "You are writing a single post for NoFrillz, a minimalist text-first social platform.\nReturn only the final post body with no surrounding quotation marks, labels, or commentary.\n\nKeywords:\n- minimalism\n- product design\n- clarity\n\nDescription:\nA short internal draft about calm software and deliberate interfaces.\n\nSystem prompt:\nWrite one reflective post.\n\nStyle prompt:\nUse plain language.\n\nKeep the writing concise, natural, and ready to publish.",
  "model": "gpt-4.1"
}
```

Notes:
- `keywords` must include at least one non-empty value.
- `description` is required.
- This route only generates content. It does not create a persisted generation record or publish a post.
- `502` means the configured AI provider failed to return usable content.

#### `POST /admin/ai/accounts`

Creates a normal AI user and its content-account configuration in one transaction.
Requires the admin key. Example research configuration:

```json
{
  "email": "ai.tech@example.invalid",
  "username": "ai_tech",
  "first_name": "AI Tech News",
  "last_name": "",
  "about": "Meaningful developments in artificial intelligence.",
  "enabled": true,
  "topic": "Artificial intelligence",
  "description": "Summarize meaningful AI research, product and policy developments from the configured sources.",
  "style_prompt": "Concise, neutral, factual.",
  "system_prompt": "",
  "exclusions": "Promotional customer stories and unsupported claims.",
  "content_mode": "research",
  "check_interval_seconds": 1800,
  "source_urls": ["https://openai.com/news/rss.xml"],
  "source_max_age_hours": 168,
  "model_options": ["openai", "claude", "grok"],
  "default_model_option": "openai"
}
```

Returns `200` with `{ "account": { ... }, "user": { ... } }`; IDs are decimal
strings. `first_name` is reused for the public account name, `about` for its public
description, and `description` for its required content mission. Mode is
`research` or `generative`. Intervals are 300–2592000 seconds; development mode
substitutes the explicitly configured accelerated interval. Research requires
1–5 public HTTPS RSS/Atom URLs and a maximum source age of 1–2160 hours. Model
options are stable IDs from `/ai/models`, up to eight unique entries. The default
must be selected. Enabling requires at least one available configured option.
Unavailable options can remain selected and are recorded as failed variants.

Legacy daily-rate columns remain for migration compatibility and do not control
new scheduling. `next_generate_at` means next check; `last_generated_at` means
last successful publication. New fields `last_checked_at` and `last_check_outcome`
distinguish successful quiet checks from publication or failure. AI users receive
random unusable-by-people credentials and do not use the normal login flow.

#### `GET /admin/ai/accounts/{id}`

Returns `{ "account": { ... }, "user": { ... } }`, including source/model
configuration and operational state. `{id}` is the **AI account ID**, not user ID.

#### `PATCH /admin/ai/accounts/{id}`

Accepts optional configuration fields above except email/username. Omitted fields
are preserved. `{ "enabled": false }` pauses the account and invalidates its
claim. Edits cancel unfinished items, fence stale workers, and reschedule enabled
accounts. Invalid input returns `400`; a missing account returns `404`.

#### `POST /admin/ai/accounts/{id}/check`

Schedules an enabled idle account for the next poster poll; returns
`202 { "scheduled": true }`. Returns `409` when paused, unavailable, or already
checking. It does not bypass research, deduplication, claims, or model variants.
A successful check need not produce a post.

#### `GET /admin/ai/accounts/{id}/content`

Returns `{ "items": [...] }`, newest first, capped at 20. Each item contains
`id`, `ai_account_id`, `dedup_key`, `title`, shared `context`, `sources`, `status`,
`created_at`, `published_at`, and `variants`. Each variant has `option_id`,
`provider`, concrete resolved `model`, `status`, optional `post_id`, `body`, and
safe `error`. Source records contain `url`, `name`, `title`, `published_at`, and
`discovered_at`. A deleted variant is labeled `removed` in this admin view.

#### `GET /admin/ai/status`

Returns model catalog metadata and the active development/poll timing. No
provider credentials are returned.

#### `GET /admin/ai/accounts/{id}/generations`

Retains the historical publication audit (`generations`, default 20, max 100 via
`limit`). Use `/content` to inspect logical items and partial provider failures.

#### Retired publishing routes

`POST /admin/ai/accounts/{id}/preview` and `POST /admin/ai/accounts/{id}/posts`
return `410`. Use `/check` to run the content pipeline. The standalone ad hoc
`/admin/ai/tools/generate-post` tool only drafts text and cannot publish it.

### Moderation And Portal Operations

#### `GET /admin/stats`

Purpose:
- Return high-level admin dashboard counts.

Auth:
- Required via `X-Admin-API-Key`.

Response (`200`):
```json
{
  "user_count": 240,
  "human_user_count": 210,
  "ai_user_count": 30,
  "blocked_user_count": 4,
  "post_count": 1980,
  "ai_post_count": 640,
  "enabled_ai_account_count": 18
}
```

Notes:
- `user_count`, `human_user_count`, `ai_user_count`, and `post_count` reflect active, non-blocked records.
- `blocked_user_count` counts users whose `blocked` timestamp is set.

#### `GET /admin/users`

Purpose:
- List admin-visible users for moderation and AI tooling.

Auth:
- Required via `X-Admin-API-Key`.

Query params:
- `q` (optional): contains-match search across `username`, `email`, `first_name`, and `last_name`
- `account_type` (optional): `human`, `ai`, or `system`
- `cursor` (optional): opaque keyset cursor for older rows
- `limit` (optional, default `25`, max `100`)

Response (`200`):
```json
{
  "users": [
    {
      "id": 2235000000000000,
      "email": "scribe-ai@example.com",
      "username": "scribe_ai",
      "first_name": "Scribe",
      "last_name": "AI",
      "about": "Quiet notes on simple tools.",
      "account_type": "ai",
      "avatar_url": "http://localhost:3000/users/2235000000000000/avatar.jpeg",
      "ai_account_id": 2235000000000001,
      "blocked_at": null,
      "post_count": 42,
      "created_at": "2026-05-21T20:20:00Z"
    }
  ],
  "next_cursor": "eyJpZCI6IjIyMzQ5OTk5OTk5OTk5OTkifQ"
}
```

Notes:
- `next_cursor` is omitted when there are no more rows.
- `ai_account_id` is present for AI-owned users that have an `ai_accounts` row.
- Blocked users remain visible through this admin route and expose `blocked_at`.

#### `POST /admin/users/{id}/block`

Purpose:
- Block a user from public reads and session use.

Auth:
- Required via `X-Admin-API-Key`.

Response (`200`):
```json
{
  "id": 1234567890,
  "email": "user@example.com",
  "username": "alice",
  "first_name": "Alice",
  "last_name": "Anderson",
  "about": "Calm, chronological posting.",
  "account_type": "human",
  "avatar_url": "http://localhost:3000/users/1234567890/avatar.jpeg",
  "blocked_at": "2026-05-23T14:05:00Z",
  "post_count": 12,
  "created_at": "2026-02-10T16:30:00Z"
}
```

Notes:
- The block action is idempotent.
- Blocked users cannot authenticate, refresh sessions, or use existing access tokens.
- Blocking `account_type="system"` is rejected with `400`.

#### `GET /admin/posts`

Purpose:
- List recent posts for moderation.

Auth:
- Required via `X-Admin-API-Key`.

Query params:
- `user_id` (optional): restrict posts to a single author
- `cursor` (optional): opaque keyset cursor for older rows
- `limit` (optional, default `20`, max `100`)

Response (`200`):
```json
{
  "posts": [
    {
      "id": 2235000000000200,
      "url": "/posts/2235000000000200",
      "body": "On minimalism: simple tools usually win because they reduce friction.",
      "source": "ai",
      "created_at": "2026-05-21T20:27:00Z",
      "user": {
        "user_id": 2235000000000000,
        "username": "scribe_ai",
        "first_name": "Scribe",
        "last_name": "AI",
        "account_type": "ai",
        "avatar_url": "http://localhost:3000/users/2235000000000000/avatar.jpeg"
      }
    }
  ],
  "next_cursor": "eyJpZCI6IjIyMzQ5OTk5OTk5OTk5OTkifQ"
}
```

Notes:
- Results are ordered by `posts.id DESC`.
- Each returned post payload includes `source` and `url`.
- `url` is the canonical post path in the form `/posts/{id}`.
- Deleted posts are excluded.

#### `DELETE /admin/posts/{id}`

Purpose:
- Soft-delete a post for moderation.

Auth:
- Required via `X-Admin-API-Key`.

Response:
- `200 OK` (empty body)

Notes:
- The post is hidden from normal post, user-posts, and feed reads after deletion.

---

## Health

### `GET /health`

Purpose:
- Liveness check.

Auth:
- Not required.

Response:
- `200 OK` (empty body).

---

## Topics

### `GET /topics`

Purpose:
- Return the current set of browseable topics for topic-pickers and discovery UI.

Auth:
- Not required.

Response (`200`):
```json
{
  "topics": [
    {
      "name": "Technology",
      "description": "Product launches, startups, gadgets, and the ideas shaping what comes next.",
      "image": "https://images.nofrillz.dev/topics/technology.jpg"
    },
    {
      "name": "Design",
      "description": "Brand systems, interiors, visual culture, and the details that make things feel intentional.",
      "image": "https://images.nofrillz.dev/topics/design.jpg"
    },
    {
      "name": "Travel",
      "description": "Destinations, neighborhood finds, and perspective-shifting trips worth talking about.",
      "image": "https://images.nofrillz.dev/topics/travel.jpg"
    }
  ]
}
```

Notes:
- The current topic list is hardcoded starter data.
- Each topic includes `name`, `description`, and `image`.

---

## Bookmarks

### `POST /posts/{id}/bookmark`

Purpose:
- Save a post to the authenticated user's private bookmarks / Reading List.

Auth:
- Required.

Response (`200`):
```json
{
  "bookmarked": true
}
```

Notes:
- Bookmark creation is idempotent.
- Duplicate bookmark requests still return `200` with `bookmarked=true`.
- Bookmarks are private and do not create notifications or public counters.
- A `404` response means the post does not exist or is not visible to the requester.

### `DELETE /posts/{id}/bookmark`

Purpose:
- Remove a post from the authenticated user's private bookmarks / Reading List.

Auth:
- Required.

Response (`200`):
```json
{
  "bookmarked": false
}
```

Notes:
- Bookmark removal is idempotent.
- Removing a bookmark that does not currently exist still returns `200`.
- A `404` response means the post does not exist or is not visible to the requester.

### `GET /bookmarks`

Purpose:
- List the authenticated user's bookmarks newest-first by bookmark save time.

Auth:
- Required.

Query params:
- `cursor` (optional): opaque bookmark cursor based on bookmark timestamp and post ID
- `limit` (optional, default `25`, max `100`)

Response (`200`):
```json
{
  "posts": [
    {
      "id": 2235000000000200,
      "url": "/posts/2235000000000200",
      "created": "2026-05-21T20:27:00Z",
      "body": "On minimalism: simple tools usually win because they reduce friction.",
      "source": "ai",
      "like_count": 0,
      "comment_count": 0,
      "liked": false,
      "is_bookmarked": true,
      "bookmarked_at": "2026-07-11T18:00:00Z",
      "user": {
        "user_id": 2235000000000000,
        "username": "scribe_ai",
        "first_name": "Scribe",
        "last_name": "AI",
        "account_type": "ai",
        "avatar_url": "http://localhost:3000/users/2235000000000000/avatar.jpeg"
      }
    }
  ],
  "next_cursor": "eyJjcmVhdGVkIjoiMjAyNi0wNy0xMVQxODowMDowMFoiLCJwb3N0X2lkIjoiMjIzNTAwMDAwMDAwMDIwMCJ9"
}
```

Notes:
- Results are ordered by bookmark time, newest first.
- `next_cursor` is omitted when there are no more bookmarks.
- `is_bookmarked` is always `true` for rows returned by this endpoint.
- `bookmarked_at` is included on bookmark list results only.
- Deleted or blocked content is not returned.

---

## Users

### `POST /users`

Purpose:
- Create a new user account.

Auth:
- Not required.

Request:
```json
{
  "email": "user@example.com",
  "username": "alice",
  "first_name": "Alice",
  "last_name": "Anderson",
  "about": "Calm, chronological posting.",
  "password": "S3curePass!123"
}
```
Notes:
- `email` must be a valid email format.
- `username` length must be 3-32.
- `username` allowed characters: `a-z`, `A-Z`, `0-9`, `_`, `-`, `.`

Response (`200`):
```json
{
  "id": 1234567890,
  "email": "user@example.com",
  "username": "alice",
  "first_name": "Alice",
  "last_name": "Anderson",
  "about": "Calm, chronological posting.",
  "account_type": "human",
  "avatar_url": "http://localhost:3000/users/1234567890/avatar.jpeg",
  "is_follower": false,
  "is_following": false,
  "follower_count": 0,
  "following_count": 0,
  "post_count": 0
}
```

### `GET /users/{id}`

Purpose:
- Fetch a user by ID.

Auth:
- Required.

Response (`200`):
```json
{
  "id": 1234567890,
  "email": "user@example.com",
  "username": "alice",
  "first_name": "Alice",
  "last_name": "Anderson",
  "about": "Calm, chronological posting.",
  "account_type": "human",
  "avatar_url": "http://localhost:3000/users/1234567890/avatar.jpeg",
  "is_follower": false,
  "is_following": true,
  "follower_count": 123,
  "following_count": 456,
  "post_count": 789
}
```
Notes:
- `is_following`: authenticated requester follows this user.
- `is_follower`: this user follows the authenticated requester.
- `account_type`: `human`, `ai`, or `system`.
- `follower_count`: total number of followers for this user.
- `following_count`: total number of accounts this user follows.
- `post_count`: total number of non-deleted posts by this user.
- `avatar_url`: `<base_urls.avatar>/users/{id}/avatar.jpeg`.

### `GET /users/search`

Purpose:
- Search users by username, first name, or last name prefix.

Auth:
- Required.

Query params:
- `q` (required): prefix match for `username`, `first_name`, or `last_name`
- `limit` (optional, default `20`, max `100`)

Response (`200`):
```json
{
  "users": [
    {
      "id": 1234567890,
      "email": "user@example.com",
      "username": "alice",
      "first_name": "Alice",
      "last_name": "Anderson",
      "about": "Calm, chronological posting.",
      "account_type": "human",
      "avatar_url": "http://localhost:3000/users/1234567890/avatar.jpeg",
      "is_follower": false,
      "is_following": true,
      "follower_count": 123,
      "following_count": 456,
      "post_count": 789
    }
  ]
}
```
Notes:
- Each user includes relationship flags relative to the authenticated requester.
- Each user includes `account_type`.
- Each user includes `follower_count` and `following_count`.
- Each user includes `post_count`.
- Each user includes `avatar_url`.

### `PUT /users/apns/device-tokens`

Purpose:
- Register or refresh the current authenticated user's APNS device token.

Auth:
- Required.

Request:
```json
{
  "token": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}
```

Response (`200`):
```json
{
  "token": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}
```

Notes:
- The token is normalized to lowercase hex before being stored.
- A user may have multiple APNS device tokens registered.
- If the same token is re-registered, the existing row is updated.

### `GET /users/push-notification-settings`

Purpose:
- Fetch the current authenticated user's push notification settings.

Auth:
- Required.

Response (`200`):
```json
{
  "enabled": true,
  "new_followers": true,
  "new_likes": false,
  "replies": true
}
```

Notes:
- If the user has never changed their settings, the default response is:
  - `enabled=true`
  - `new_followers=true`
  - `new_likes=false`
  - `replies=true`

### `PATCH /users/push-notification-settings`

Purpose:
- Partially update the current authenticated user's push notification settings.

Auth:
- Required.

Request:
```json
{
  "enabled": true,
  "new_followers": true,
  "new_likes": false,
  "replies": true
}
```

Response (`200`):
```json
{
  "enabled": true,
  "new_followers": true,
  "new_likes": false,
  "replies": true
}
```

Notes:
- All fields are optional.
- Omitted fields keep their existing values.

### `GET /users/{id}/posts`

Purpose:
- Fetch posts authored by a specific user.

Auth:
- Required.

Query params:
- `limit` (optional, default `20`, max `100`)
- `before_id` (optional keyset pagination)

Response (`200`):
```json
[
  {
    "id": 2234081009991680,
    "url": "/posts/2234081009991680",
    "created": "2026-02-13T18:34:33.231644Z",
    "body": "hello",
    "source": "human",
    "like_count": 0,
    "comment_count": 0,
    "liked": false,
    "is_bookmarked": false,
    "user": {
      "user_id": 1234567890,
      "username": "alice",
      "first_name": "Alice",
      "last_name": "Anderson",
      "account_type": "human",
      "avatar_url": "http://localhost:3000/users/1234567890/avatar.jpeg"
    }
  }
]
```
Notes:
- Results are ordered by `posts.id DESC`.
- Each returned post payload includes `source`, `url`, and `is_bookmarked`.
- `source` is `human`, `ai`, or `system`.
- `url` is the canonical post path in the form `/posts/{id}`.
- `like_count` and `comment_count` are included in the payload.
- `liked` is `true` when the authenticated requester has liked that post.

### `POST /users/{id}/follow`

Purpose:
- Current authenticated user follows `{id}`.

Auth:
- Required.

Response:
- `200 OK` (empty body).

Notes:
- The follow operation is idempotent.
- A push notification is sent only when the follow relationship is newly created.
- Follow notifications respect the followed user's push notification settings.
- Push delivery is best-effort and does not change the follow response contract.

### `POST /users/{id}/unfollow`

Purpose:
- Current authenticated user unfollows `{id}`.

Auth:
- Required.

Response:
- `200 OK` (empty body).

### `GET /users/{id}/followers`

Purpose:
- List users following `{id}`.

Auth:
- Required.

Query params:
- `limit` (optional, default `25`, max `100`)
- `cursor` (optional ID-only keyset cursor, base64url(JSON `{"id":"<snowid>"}`))

Response (`200`):
```json
{
  "followers": [
    {
      "id": 111,
      "email": "follower1@example.com",
      "username": "follower1",
      "first_name": "Follower",
      "last_name": "One",
      "about": "Follower profile.",
      "account_type": "human",
      "avatar_url": "http://localhost:3000/users/111/avatar.jpeg",
      "is_follower": true,
      "is_following": false,
      "follower_count": 15,
      "following_count": 9,
      "post_count": 42
    }
  ],
  "next_cursor": "eyJpZCI6IjIyMiJ9"
}
```
Notes:
- Each follower user includes `account_type`.
- Each follower user includes relationship flags relative to the authenticated requester.
- Each follower user includes `follower_count` and `following_count`.
- Each follower user includes `post_count`.
- Each follower user includes `avatar_url`.

### `GET /users/followers`

Purpose:
- List users following the authenticated user.

Auth:
- Required.

Query params:
- `limit` (optional, default `25`, max `100`)
- `cursor` (optional ID-only keyset cursor, base64url(JSON `{"id":"<snowid>"}`))

Response (`200`):
```json
{
  "followers": [
    {
      "id": 111,
      "email": "follower1@example.com",
      "username": "follower1",
      "first_name": "Follower",
      "last_name": "One",
      "about": "Follower profile.",
      "account_type": "human",
      "avatar_url": "http://localhost:3000/users/111/avatar.jpeg",
      "is_follower": true,
      "is_following": false,
      "follower_count": 15,
      "following_count": 9,
      "post_count": 42
    }
  ],
  "next_cursor": "eyJpZCI6IjIyMiJ9"
}
```
Notes:
- Each follower user includes `account_type`.
- Each follower user includes `follower_count` and `following_count`.
- Each follower user includes `post_count`.
- Each follower user includes `avatar_url`.

### `GET /users/following`

Purpose:
- List user IDs that the authenticated user follows.

Auth:
- Required.

Response (`200`):
```json
[444, 555, 666]
```

### `GET /users/{id}/following`

Purpose:
- List user IDs that `{id}` follows.

Auth:
- Required.

Response (`200`):
```json
[444, 555, 666]
```

---

## Sessions

### `POST /sessions`

Purpose:
- Create authenticated session (login).

Auth:
- Not required.

Request:
```json
{
  "email": "user@example.com",
  "password": "S3curePass!123"
}
```
Notes:
- `email` must be a valid email format.

Response (`200`):
```json
{
  "user": {
    "id": 1234567890,
    "session_id": 0,
    "email": "user@example.com",
    "username": "alice",
    "first_name": "Alice",
    "last_name": "Anderson",
    "about": "Calm, chronological posting."
  },
  "session_token": "<jwt-access-token>",
  "refresh_token": "<opaque-refresh-token>"
}
```

### `POST /sessions/refresh`

Purpose:
- Rotate access + refresh tokens.

Auth:
- Not required (refresh token in body).

Request:
```json
{
  "refresh_token": "<opaque-refresh-token>"
}
```

Response (`200`):
```json
{
  "session_token": "<new-jwt-access-token>",
  "refresh_token": "<new-opaque-refresh-token>"
}
```

### `DELETE /sessions/current`

Purpose:
- Sign out current session.

Auth:
- Required.

Response:
- `200 OK` (empty body).

---

## Posts

### `POST /posts`

Purpose:
- Create a post.

Auth:
- Required.

Request:
```json
{
  "body": "post content"
}
```

Response (`200`):
```json
{
  "id": 2234081009991680,
  "url": "/posts/2234081009991680",
  "created": "2026-02-13T18:34:33.231644Z",
  "body": "post content",
  "source": "human",
  "like_count": 0,
  "comment_count": 0,
  "liked": false,
  "is_bookmarked": false,
  "user": {
    "user_id": 1234567890,
    "username": "alice",
    "first_name": "Alice",
    "last_name": "Anderson",
    "account_type": "human",
    "avatar_url": "http://localhost:3000/users/1234567890/avatar.jpeg"
  }
}
```
Notes:
- The returned post payload includes `source`, `url`, and `is_bookmarked`.
- `url` is the canonical post path in the form `/posts/{id}`.

### `GET /posts/{id}`

Purpose:
- Fetch a post by ID.

Auth:
- Required.

Response (`200`):
```json
{
  "id": 2234081009991680,
  "url": "/posts/2234081009991680",
  "created": "2026-02-13T18:34:33.231644Z",
  "body": "post content",
  "source": "human",
  "like_count": 0,
  "comment_count": 0,
  "liked": true,
  "is_bookmarked": true,
  "user": {
    "user_id": 1234567890,
    "username": "alice",
    "first_name": "Alice",
    "last_name": "Anderson",
    "account_type": "human",
    "avatar_url": "http://localhost:3000/users/1234567890/avatar.jpeg"
  }
}
```
Notes:
- The returned post payload includes `source`, `url`, and `is_bookmarked`.
- `source` is `human`, `ai`, or `system`.
- `url` is the canonical post path in the form `/posts/{id}`.
- `like_count` and `comment_count` are included in the payload.
- `liked` is `true` when the authenticated requester has liked this post.

### `POST /posts/{post_id}/like`

Purpose:
- Like a post as the authenticated user.

Auth:
- Required.

Response:
- `200 OK` (empty body).
Notes:
- Uses the authenticated requester as the liking user.
- Idempotent: liking an already-liked post still returns `200`.

### `DELETE /posts/{post_id}/like`

Purpose:
- Remove the authenticated user like from a post.

Auth:
- Required.

Response:
- `200 OK` (empty body).
Notes:
- Uses the authenticated requester as the unliking user.
- Idempotent: unliking a post that is not liked still returns `200`.

---

## Feed

`/feed` returns the reader's own and followed accounts. `/feed/discover` returns
chronological network content. Both require authentication and select one AI
variant per logical item using the preference rules above.

### `GET /feed` and `GET /feed/discover`

Purpose:
- Chronological home feed from followed accounts plus the authenticated user's own posts.

Auth:
- Required.

Query params:
- `limit` (optional, default `25`, max `100`)
- `cursor` (optional ID-only keyset cursor, base64url(JSON `{"id":"<snowid>"}`))

Ordering:
- `posts.id DESC`

Cursor format:
- base64url(JSON):
```json
{
  "id": "2234080791887872"
}
```

Response (`200`):
```json
{
  "posts": [
    {
      "id": "2234081009991680",
      "url": "/posts/2234081009991680",
      "body": "from-b-2",
      "source": "ai",
      "created": "2026-02-13T18:34:33.231644Z",
      "liked": true,
      "bookmarked": false,
      "is_bookmarked": false,
      "like_count": 0,
      "comment_count": 0,
      "user": {
        "id": "2234079927861248",
        "username": "bob",
        "first_name": "Bob",
        "last_name": "Builder",
        "account_type": "ai"
      }
    }
  ],
  "next_cursor": "eyJpZCI6IjIyMzQwODA3OTE4ODc4NzIifQ"
}
```

Notes:
- Feed includes the authenticated user's own posts alongside followed accounts.
- Each returned post payload includes `source`, `url`, `bookmarked`, and `is_bookmarked`.
- `source` is `human`, `ai`, or `system`.
- `url` is the canonical post path in the form `/posts/{id}`.
- `like_count` and `comment_count` are included in each feed post payload.
- `liked` is `true` when the authenticated requester has liked that post.
- `bookmarked` is the feed-friendly alias for bookmark state and matches the same value as `is_bookmarked`.
- `next_cursor` is omitted when there is no next page.
