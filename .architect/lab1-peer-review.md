# Lab 1 Peer Review — Architect Notes (Planted Regression)

**DO NOT SHARE WITH PARTICIPANTS.**

## What's planted

The `lab1/peer-review` branch ships a complete, working FBT implementation
that builds and runs. It deliberately drops `cartservice.GetCartHistory` from
the design — a defensible scope decision compared to `lab1/complete`.

**The bug lives in `src/recommendationservice/fbt.py`:**

```python
ranked = sorted(totals.items(), key=lambda kv: kv[1], reverse=True)
ranked = ranked[:max_results - 1]   # off-by-one
```

The slice should be `ranked[:max_results]`. With the off-by-one, callers
asking for 4 items get 3. Default behavior (`max_results=4` from the
frontend) returns 3 items instead of 4.

## Why it's hard to catch

- Builds clean across all three services.
- Runs end-to-end without errors.
- Cart page renders the FBT panel with real-looking suggestions.
- Engineers won't *count* the items returned unless they're looking for it.
- The bug doesn't manifest as a stack trace or warning; it's a quiet
  off-by-one in product output.

## Hints to surface in pod when needed

In rough escalation order:

1. "How many items do you expect on the cart page? How many do you see?"
2. "What does max_results control end-to-end? Walk it from the proto request
    to the slice."
3. "Read fbt.py line by line. Compare what it does to what the docstring
    says it should do."

If a pod still misses it after hint 3, drop in to point at the slice.

## Reasonable false positives engineers may surface

These are *not* the planted regression but worth discussing if engineers
flag them — they're judgment calls, not bugs:

- The `top_seed[cand]` assignment uses an awkward inequality
  (`if count > totals.get(cand, 0) - count:`). It works, but is unusual.
  Document it as "I'd refactor that for clarity but it's not the bug."
- `compute_fbt` calls `name_resolver(seed)` per-item — could batch. Not a
  correctness bug, just a perf observation.
- No cart history personalization. This is the design *delta* vs.
  `lab1/complete`, not a bug. Engineers who flag it should be redirected
  to "is that wrong, or just a different scope decision?"

## Verifying the find

When an engineer locates the line, confirm:

- They can articulate what `max_results - 1` does vs. what was intended.
- They can describe the symptom (returns one fewer than requested).
- Their `notes/regression.md` includes the file, line, and the prompt that
  surfaced it.

A pod that found it via Claude reading the diff and asking "what else?"
counts. A pod that found it by counting items in the browser also counts —
whichever path got them there.
