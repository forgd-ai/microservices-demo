# Regression — culprit commit

**Commit:** `23ae4af` — `refactor: extract recommendation orchestration into helper`

## Why

The commit moves orchestration from the gRPC handler into the new
`recommendation_helpers.compute_for_user`. Most of the diff is honest
code movement, but the new module also introduces a per-user FBT
response cache:

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

The cache key is `request.user_id` alone. `request.product_ids` (the
caller's current cart contents) is not part of the key. Within the 30s
TTL, two FBT calls for the same user but different carts return the
same response — the customer-reported staleness.

## How we found it

Sub-agents in parallel narrowed the suspect to recommendationservice
(see `notes/investigation.md`). `git log --oneline -- src/recommendationservice/recommendation_helpers.py`
shows the helper landed in this single commit. Reading the diff
surfaces the cache.
