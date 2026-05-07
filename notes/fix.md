# Fix — drop the per-user FBT response cache

## What changed

`recommendation_helpers.compute_for_user` no longer caches its
response. The `_response_cache` dict, the `_RESPONSE_TTL_SECONDS`
constant, the `time` import, and the cache lookup at the top of
`compute_for_user` are gone. Behaviour is otherwise identical.

## Why this fix

Considered four options for closing the cart-recommendation staleness:

- **Drop the cache.** *Chosen.* Smallest diff, removes the bug
  entirely. The original premise of the cache — "the FBT path is hot,
  catalog round-trips add up" — is real but already mitigated by the
  frontend's in-flight dedup (`fbtInFlightCalls`), which is the
  redundancy the cache was nominally addressing. Without that
  pre-existing dedup the trade-off would be tighter.
- **Include cart contents in the cache key.** Works, preserves the
  cache. Rejected because the staleness was symptomatic of a deeper
  contract problem: the cache silently assumed cart-content changes
  weren't worth invalidating on. Including the cart in the key fixes
  *this* staleness but leaves the rec service's caching contract
  fragile against future cart-mutating signals (e.g. checkout, item
  removal). Drop-the-cache is more defensible.
- **Reduce TTL to ~1 second.** Rejected. Doesn't fix the race; just
  narrows it. A "feels right" hypothesis that doesn't survive scrutiny.
- **Have cartservice publish invalidation events to recommendationservice.**
  Architecturally correct shape if we wanted to keep the cache. Adds a
  service-to-service dependency and a new contract to maintain. Over-
  engineered for the load levels here; revisit if FBT becomes a real
  hotspot.

## Verification

`scripts/repro-fbt-bug.sh` exits 0 with `===> No bug observed: FBT
response updated after cart change.` against this branch. With the
cache restored (revert the fix), it exits 1 reproducibly. Regression
test in the next commit.
