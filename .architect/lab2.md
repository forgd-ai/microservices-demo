# Lab 2 — Architect Notes (Distributed Debug)

**DO NOT SHARE WITH PARTICIPANTS.**

## The bug

Per-user FBT response cache in
`src/recommendationservice/recommendation_helpers.py`,
inside `compute_for_user`. Keyed only on `request.user_id`, TTL 30
seconds. `request.product_ids` (the caller's current cart) is **not**
part of the key, so cart-content changes within the TTL window return
the prior cached response.

```python
_RESPONSE_TTL_SECONDS = 30
_response_cache = {}  # user_id -> (timestamp, FBTResponse)

def compute_for_user(request, ...):
    cached = _response_cache.get(request.user_id)
    if cached is not None:
        cached_at, cached_response = cached
        if time.time() - cached_at < _RESPONSE_TTL_SECONDS:
            return cached_response
    ...
```

The fix-direction ranges over four options (drop the cache; include
cart contents in the key; reduce/eliminate TTL; cartservice publishes
invalidation events). All are defensible; canonical lab2/checkpoint-fix
drops the cache.

## The introducing commit

`23ae4af refactor: extract recommendation orchestration into helper`.

The commit is mostly honest code movement — the FBT orchestration
moves from `recommendation_server.ListFrequentlyBoughtTogether` into
the new `recommendation_helpers.compute_for_user`. The cache is added
as part of that refactor, presented as a routine perf optimisation
inside the new helper, and not mentioned in the commit message. The
diff is large enough that the cache is easy to miss on a first pass.

## Burying

Five commits sit between the bug commit and `lab2/start` HEAD, in
order from oldest to newest:

1. `14b0dae test: add sanity checks for FBT seed co-occurrence data`
2. `23ae4af refactor: extract recommendation orchestration into helper` ← **BUG**
3. `622b3fd cartservice: switch redis store logs to compact key=value format`
4. `19765b7 frontend: extract cart-line view assembly into a helper`
5. `5c6b5fb docs: annotate FBT seed data with product cluster groupings`
6. `7e67483 frontend: dedupe FBT requests on rapid cart-page navigation` ← **red herring**
7. `420aa09 add scripts/repro-fbt-bug.sh for Lab 2 reproduction`

The frontend dedup at HEAD is the most plausible suspect on read —
its title is adjacent to the customer report ("recommendations don't
update" + "rapid navigation"). The cartservice logging change is also
suspicious to a careful investigator. Both are benign. The bug is in
the unsexy "extract orchestration" commit two services over from the
visible symptom.

## Symptoms vs hypothesis

A correct hypothesis must explain three things:

- **Cross-service.** The bug only manifests when cart contents change
  between FBT calls — i.e., cartservice and recommendationservice
  both in the loop. Single-service tests of either pass.
- **First reload wrong.** The first reload after a cart change hits
  the existing cache entry.
- **Eventually catches up.** After 30 seconds, the next call rebuilds
  the cache from current request inputs. The "refreshes" don't fix
  it — *waiting* does.

If a pod's hypothesis only explains some, push them back to the file
and ask "what would have to be true for that to be the whole story?"

## Sub-agent briefs

Engineers should spawn one investigator per service. A vague brief
yields a vague report. Tight briefs that work:

- **cartservice** — "Does AddItem write through synchronously? Could a
  recent cart change be invisible to a subsequent GetCart for the
  same user?"
- **recommendationservice** — "Is there any caching, memoization, or
  per-user state that survives across requests? If so, what does the
  cache key include?"
- **frontend** — "Does the cart-page handler call FBT with the
  pre-update or post-update cart? Trace the read order. Does the
  fbtInFlightCalls dedup affect responses across distinct carts?"

If pods stall, the recommendationservice brief is the highest-yield
to sharpen.

## Valid fixes

All four work; there isn't one right answer. Engineers should
articulate the trade-off they chose:

1. **Drop the cache.** Canonical answer in `lab2/checkpoint-fix`.
   Smallest diff. The frontend's existing in-flight dedup
   (`fbtInFlightCalls`) already eliminates the redundant catalog
   round-trip the cache nominally addressed.
2. **Include cart contents in the cache key.** Preserves the cache,
   smaller diff than (3). Still leaves the contract fragile against
   future cart-mutating signals (item removal, checkout) that don't
   surface in `request.product_ids`.
3. **Reduce TTL to ~1 second.** Doesn't fix the race, narrows it.
   Reject this — push back on engineers who land here.
4. **cartservice publishes invalidation events.** Architecturally
   correct shape if the cache is worth keeping. Adds a service-to-
   service dependency for a problem that doesn't justify it at this
   scale. Over-engineered for the workshop scenario.

Engineers landing on (1) or (2) should be able to defend the choice.
(3) is a "feels right" hypothesis — challenge it. (4) earns credit
for the idea, then asks "what's the smallest change that fixes it?"

## Regression test

The canonical test (`lab2/complete:src/recommendationservice/tests/test_fbt_response_freshness.py`)
calls `compute_for_user` directly with stubbed cart and catalog stubs
and asserts two calls for the same user with different carts produce
different items. Verified failing on `lab2/start`, passing on
`lab2/checkpoint-fix`.

Engineers may write integration tests instead — fine, arguably more
realistic. What matters is verifying both directions: the test fails
without the fix, passes with it. A test that can't catch the
regression is theatre.

## Hints to surface in pod when needed

In rough escalation order:

1. "What does each of your three sub-agents say? Which report
   contained the most surprising thing?"
2. "Where in the system would two requests with different inputs
   return the same output?"
3. "Look at `git log --oneline -- src/recommendationservice/recommendation_helpers.py`.
   The helper landed in a single commit. Read the diff."

If a pod still can't see it after hint 3, drop in to point at the
cache lookup.

## Reproduction

`scripts/repro-fbt-bug.sh` is the deterministic repro. It runs
inside the recommendationservice container, drives cart and rec
services via gRPC, and exits non-zero on bug detection. Useful as a
"did your fix actually fix it?" gate.
