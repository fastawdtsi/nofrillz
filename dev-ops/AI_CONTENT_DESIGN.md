# Content account correction

Keep: the normal AI user identity/badge, shared posts service, MySQL migrations,
claim tokens and stale-claim recovery, bounded scheduler loop, failure isolation,
OpenAI client, provider interface, Compose, admin authentication and account CRUD.

Modify: an account describes a content mission and either research or generative
mode. Its interval schedules checks, not a publishing quota. A quiet research
check succeeds without posting. Research sources are explicit RSS/Atom URLs;
retrieval is bounded and separate from model generation.

Add: logical content items with canonical source fingerprints and provenance;
model variants linking to ordinary posts; stable logical model options mapped by
server configuration to provider/model identifiers; global preferences and an
optional override on the existing follow relationship. Feed selection ranks
successful variants by override, global preference, account default, then option
ID. Cursor ordering uses a stable logical item ID. Likes/bookmarks remain tied to
the specific variant a user actually read; bookmarks and direct links preserve it.

Remove: instructions to invent personal experiences or imitate fictional people,
and the misleading persona/daily-quota UI. Retire the three local fictional demo
accounts after the replacement workflow is proven; preserve other data.

Cost/recovery: one retrieved context per research item; one shared generative
seed per evergreen item. Persist completed variants and skip them on recovery.
Each configured option is attempted once per item, with failure recorded and
successful variants retained. A subsequent scheduled item can try that provider
again; whole-cycle failures retain exponential backoff. No automatic retry storm
for failed variants. A stale worker cannot persist after its claim changes.

Research validation: live inspection found plausible-sounding additions beyond a
short source summary. The writer prompt now explicitly excludes inferred goals
and benefits, and a separate source-evidence review gates each research variant
before publication. It uses the available account default model (otherwise the
writer). Rejected drafts remain failed variants; other successful versions are
retained. This bounded extra call is for factual discipline, not more content.
The check is probabilistic and does not replace source quality or moderation.

Generative consistency: a small model sometimes replaced the accepted joke with
an unrelated one. Alternative evergreen variants now receive a seed-consistency
review before publishing. The default reviewer falls back to the writer on an
outage; a failed provider cannot prevent the others from producing valid variants.
Initial unreviewed compact-model demo variants were soft-deleted, retaining their
audit rows and the original accepted seeds.

Research editorial review also receives the account mission and exclusions.
An excluded or insignificant source returns `__NO_POST__`, records a successful
`not_significant` check, and avoids calls for remaining variants. A dedicated
MySQL test verifies this behavior; exclusions are not treated as provider failures.

Novelty: publication now checks the account's entire stored post history across
all models, including legacy and soft-deleted posts. Case/punctuation-normalized
identical text and identical research context are suppressed deterministically.
The current logical item is excluded so its sibling model variants remain valid.
Generation receives the last 15 distinct items across models, not just output
from the current model.

An additional editorial review compares each validated draft with up to 12
lexically closest historical items plus four recent items. It rejects recycled
facts, tips, jokes and creative tasks even when wording, headline or source URL
changes, while allowing substantive new developments on an existing topic.
The query scans all retained history; only the model's comparison shortlist is
bounded. Candidate bodies and context are bounded too. No vector service, new
credentials, schema migration, or per-topic code is required.

A repeat records a `duplicate` variant with its previous post/item reference;
if no variant has been published yet, the item is skipped, the account records
a successful `duplicate` check, and remaining model calls stop. No immediate
regeneration occurs. Recovery preserves that decision. If an alternative for an
otherwise new item repeats older output, only that alternative is suppressed.
Reviewer failures, uncertain decisions and malformed/unknown references cannot
publish unchecked content; normal provider failure handling and backoff apply.

Scope and limits: deduplication is per content account. Different accounts may
legitimately cover the same development for different missions. Exact comparison
has no age/count window while the posts remain stored. Semantic matching remains
probabilistic: an old paraphrase with little shared vocabulary can miss the
bounded shortlist, and an LLM can misclassify. Hard deletion removes that portion
of the history. This is stronger prevention, not a mathematical uniqueness
guarantee. The current streaming scan uses bounded memory but work grows with
account history; an indexed retrieval layer can replace it if profiling warrants.
