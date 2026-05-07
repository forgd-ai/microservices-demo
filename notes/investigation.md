# Investigation — FBT staleness across cart-recommendation boundary

## Sub-agent findings (parallel investigation)

Three investigators ran in parallel — one per service.

### cartservice — clean

- `AddItem` / `EmptyCart` / `GetCart` / `GetCartHistory` write through
  to Redis synchronously. Logs confirm each operation.
- A separate per-user history list is appended on every `AddItem` and
  exposed via `GetCartHistory`. Reads return current state.
- Recent commits to this service (the redis store logging change) are
  cosmetic — same operations, different log format.

Conclusion: cartservice is not the source.

### frontend — clean

- `viewCartHandler` reads the cart fresh on every render, then calls
  both `getRecommendations` and `getFrequentlyBoughtTogether` with the
  current cart contents.
- `cartIDs(cart)` is computed *after* the cart fetch, so the FBT
  request always carries up-to-date product ids.
- A recent dedup wrapper (`fbtInFlightCalls`) coalesces concurrent
  calls with identical inputs but doesn't cache across distinct cart
  contents — keys include the sorted product id list.

Conclusion: the request leaving the frontend is correct. The bug is
not in dedup, not in the handler order, not in the template.

### recommendationservice — suspect

- `ListFrequentlyBoughtTogether` delegates to
  `recommendation_helpers.compute_for_user`.
- `compute_for_user` consults a module-level dict keyed only on
  `request.user_id`. On hit within `_RESPONSE_TTL_SECONDS` (30s), it
  returns the cached `FBTResponse` without recomputing.
- The cache key does **not** include the cart contents
  (`request.product_ids`).

This is the gap. The frontend correctly tells the recommendation
service the latest cart, the cart service correctly stores it — but
the recommendation service hands back a response that was computed
under a previous cart, because nothing in the cache key reflects the
cart change.

## Hypothesis

The bug lives at the cart-recommendation boundary, not inside either
service:

- cartservice writes are correct (sub-agent verified).
- recommendationservice computation is correct *given the inputs it
  honours* — but it ignores the new cart contents in favour of a
  cached response keyed only on `user_id`.
- The TTL-based eviction is the only thing that breaks the staleness,
  which is why "refreshing a few times" appears to fix it (it doesn't —
  *waiting* does).

## Why every symptom matches

- **Cross-service:** the bug only shows when cart contents change
  between FBT calls. Single-service tests of cartservice or
  recommendationservice in isolation never exercise this.
- **First reload wrong:** the first reload after a cart change hits
  the existing cache entry.
- **Eventually catches up:** after 30 seconds, the next call rebuilds
  the cache from current request inputs.
- **Single-service tests pass:** there is no test that drives
  `ListFrequentlyBoughtTogether` twice for the same user with
  different cart contents within the TTL window — that combination
  is the only one that surfaces the staleness.

## Suspect commits to walk

`git log --oneline -- src/recommendationservice/recommendation_server.py src/recommendationservice/recommendation_helpers.py`
shows the recent activity in this area. The orchestration was extracted
into a helper recently — the commit that introduced
`recommendation_helpers.py` is where the cache lives.
