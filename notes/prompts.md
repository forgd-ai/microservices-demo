# Build conversation — reference/fbt-cooccurrence

This branch shows what FBT looks like when you ask Claude for a more
sophisticated implementation than "hardcoded co-occurrence pairs."
The goal is not the algorithm — it's the prompt scaffolding that
produced it.

## Conversation skeleton

What the architect actually said to Claude, edited down to the
essentials. Verbatim where it matters; paraphrased where the wording
is incidental.

### 1. Frame the upgrade, not the task

> The current FBT impl uses a hardcoded co-occurrence dict in
> `fbt_data.py` keyed on product_id with (paired_id, count) tuples.
> That works but doesn't reflect what FBT actually means in retail.
> We want to replace it with a real association-mining approach over a
> seeded transaction history. Compute lift and confidence; rank by
> lift; surface the metric in the response. Don't change the proto
> contract — keep the existing FBT messages.

Plan-mode discussion. Claude asked whether the transactions should be
loaded once at startup or per-request. Architect chose per-request to
keep the diff small and the code transparent (acknowledging the perf
cost as a follow-up).

### 2. Ask for the algorithm before the integration

> Before touching `recommendation_server.py`, write a self-contained
> module `fbt_cooccurrence.py` that takes a list of transactions and
> a seed list and returns ranked candidates with lift/confidence/count.
> Include docstrings explaining support, confidence, and lift in
> retail terms. Pure functions; no I/O except loading transactions.

This separation matters. It made the next step a small wiring change
rather than a 200-line replacement.

### 3. Seed the transaction file from the catalog, not from imagination

> The catalog has 9 products. Create `transactions.json` with ~25-30
> baskets that produce sensible FBT pairings — apparel items
> (Sunglasses, Tank Top, Watch, Loafers) cluster together; home goods
> (Candle Holder, Salt & Pepper, Mug, Bamboo Jar) cluster together;
> Hairdryer is a bridge. Validate by hand: lift for Sunglasses → Watch
> should be > 1; lift for Sunglasses → Mug should be ~ 0.

After Claude generated the file, architect ran a quick eyeball check
in REPL: `compute_fbt(['OLJCESPC7Z'], ['OLJCESPC7Z'], 4)`. Lift values
came out ~ 2.0 for the apparel pairings, near zero for cross-cluster.

### 4. Wire only what changes

> In `recommendation_server.py`, change the import from `fbt_data`
> to `fbt_cooccurrence` and replace the inline scoring loop with a
> call to `compute_fbt`. Pull the lift value into the reason text:
> "Often bought with X (lift 2.3)". Everything else (cart history
> fetch, name resolution, response building) stays.

One-shot change. Reviewed the diff before accepting. Claude got it
right on the first try because the function signature was clear.

### 5. Sanity-check end-to-end

> Run `compute_fbt` directly from a Python REPL with a few seed carts.
> Print the top results. Confirm cross-cluster recommendations are
> rare and apparel-clusters with apparel.

That's the final validation step. The REPL is faster than spinning
up the stack and clicking through the UI.

## What's worth taking from this

- **The architect did the design thinking.** Claude wrote the code,
  but the choice of algorithm (lift over cooccurrence count), the
  shape of the seed file, and the validation criteria came from the
  architect.
- **Modules first, integration second.** Asking for a self-contained
  pure-function module before touching the gRPC handler made the
  integration a 30-line diff instead of a 200-line one.
- **Validation criteria, not just "make it work".** "Lift for
  Sunglasses → Watch should be > 1" is testable in 10 seconds in a
  REPL. "Make FBT better" is not.
- **No new proto fields needed.** Claude initially proposed adding
  `lift` and `confidence` fields to `FBTItem`. Architect pushed back —
  encoding them in the existing `reason` text fits the contract and
  saves a round of stub regeneration.

## What this is *not*

This is not a production FBT system. The transactions seed file is
synthetic, association mining recomputes per-request, and the lift
threshold isn't tuned. For a real implementation you'd:

- Mine associations from the actual transaction warehouse
- Precompute support and pair-support tables, refresh nightly
- Add a min-support filter to drop noise pairs
- Consider cluster-of-clusters (apparel vs. home) as a categorical
  prior

For the workshop, the point is showing that "more sophisticated than
the canonical answer" is reachable in a focused conversation with
clear validation criteria — not that this specific implementation is
the right one for production.
