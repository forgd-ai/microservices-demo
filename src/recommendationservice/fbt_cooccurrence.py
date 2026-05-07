"""Co-occurrence FBT scoring with lift and confidence.

Reference implementation for the workshop. Loads a synthetic transaction
history from transactions.json and ranks candidates by lift over a
seed cart, with confidence as a tiebreaker.

Definitions:
  support(X)    = P(X in basket)         = |baskets containing X| / N
  confidence(A→B) = P(B in basket | A in basket)
                  = |baskets with A and B| / |baskets with A|
  lift(A→B)     = confidence(A→B) / support(B)

Lift > 1 means A and B co-occur more than chance; we rank by lift so
that "frequently bought together" actually reflects pairs that go
together more than they should at random — not just popular pairs.

This is a deliberately simple implementation: O(transactions × seeds)
per query, recomputed each time. For production you'd precompute the
support and pair-support tables once at boot. Kept transparent here so
the algorithm reads cleanly.
"""

import json
import math
from collections import Counter
from pathlib import Path


_DEFAULT_PATH = Path(__file__).resolve().parent / "transactions.json"


def load_transactions(path=None):
    """Load the transaction history from JSON."""
    p = Path(path) if path else _DEFAULT_PATH
    with p.open() as f:
        data = json.load(f)
    return [list(basket) for basket in data["transactions"]]


def _basket_counts(transactions):
    """Per-product basket count and total basket count."""
    item_counts = Counter()
    for basket in transactions:
        for item in set(basket):
            item_counts[item] += 1
    return item_counts, len(transactions)


def _pair_counts(transactions, seeds):
    """For each seed, count baskets containing seed AND each other product."""
    pair_counts = {seed: Counter() for seed in seeds}
    seed_basket_count = Counter()
    for basket in transactions:
        items = set(basket)
        for seed in seeds:
            if seed in items:
                seed_basket_count[seed] += 1
                for other in items:
                    if other != seed:
                        pair_counts[seed][other] += 1
    return pair_counts, seed_basket_count


def compute_fbt(seed_ids, exclude_ids, max_results, transactions=None):
    """Rank candidate products by aggregated lift across the seed cart.

    seed_ids: iterable of product ids (current cart contents)
    exclude_ids: iterable of product ids to omit from the result
    max_results: max items to return (defaults to 4 when <= 0)
    transactions: optional override for testing; loads from disk by default

    Returns a list of dicts: {product_id, lift, confidence, count, top_seed}
    sorted by aggregated lift descending. count is the raw basket
    co-occurrence with top_seed (the seed item that contributed the most
    lift to this candidate).
    """
    if max_results <= 0:
        max_results = 4
    seeds = list(seed_ids)
    excludes = set(exclude_ids) | set(seeds)

    if transactions is None:
        transactions = load_transactions()
    if not transactions:
        return []

    item_counts, total = _basket_counts(transactions)
    pair_counts, seed_basket_count = _pair_counts(transactions, seeds)

    aggregated = {}    # candidate -> aggregated lift across seeds
    best_pair = {}     # candidate -> (seed, lift, conf, count) of top contributor
    for seed in seeds:
        seed_n = seed_basket_count.get(seed, 0)
        if seed_n == 0:
            continue
        for cand, pair_n in pair_counts[seed].items():
            if cand in excludes:
                continue
            cand_support = item_counts[cand] / total
            confidence = pair_n / seed_n
            if cand_support == 0:
                continue
            lift = confidence / cand_support

            aggregated[cand] = aggregated.get(cand, 0.0) + lift
            existing = best_pair.get(cand)
            if existing is None or lift > existing[1]:
                best_pair[cand] = (seed, lift, confidence, pair_n)

    ranked = sorted(aggregated.items(), key=lambda kv: kv[1], reverse=True)[:max_results]
    out = []
    for cand, _ in ranked:
        seed, lift, confidence, count = best_pair[cand]
        out.append({
            "product_id": cand,
            "lift": lift,
            "confidence": confidence,
            "count": count,
            "top_seed": seed,
        })
    return out
