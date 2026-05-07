"""Frequently bought together logic.

Pure function over a co-occurrence map; no I/O. Kept separate from
recommendation_server.py so the gRPC handler stays thin and the FBT
computation can be unit-tested without bringing up gRPC.
"""

import demo_pb2

# Seed co-occurrence data: product_id -> [(paired_product_id, count), ...]
# sorted by count descending. Counts are notional basket co-occurrences from
# a synthetic transaction history.
_COOCCURRENCE = {
    "OLJCESPC7Z": [("66VCHSJNUP", 24), ("1YMWWN1N4O", 18), ("L9ECAV7KIM", 12)],
    "66VCHSJNUP": [("OLJCESPC7Z", 24), ("L9ECAV7KIM", 15), ("2ZYFJ3GM2N", 8)],
    "1YMWWN1N4O": [("L9ECAV7KIM", 22), ("OLJCESPC7Z", 18), ("66VCHSJNUP", 11)],
    "L9ECAV7KIM": [("1YMWWN1N4O", 22), ("66VCHSJNUP", 15), ("OLJCESPC7Z", 12)],
    "2ZYFJ3GM2N": [("0PUK6V6EV0", 14), ("LS4PSXUNUM", 10), ("66VCHSJNUP", 8)],
    "0PUK6V6EV0": [("LS4PSXUNUM", 28), ("9SIQT8TOJO", 19), ("2ZYFJ3GM2N", 14)],
    "LS4PSXUNUM": [("0PUK6V6EV0", 28), ("6E92ZMYYFZ", 21), ("9SIQT8TOJO", 16)],
    "9SIQT8TOJO": [("0PUK6V6EV0", 19), ("LS4PSXUNUM", 16), ("6E92ZMYYFZ", 13)],
    "6E92ZMYYFZ": [("LS4PSXUNUM", 21), ("9SIQT8TOJO", 13), ("0PUK6V6EV0", 9)],
}


def compute_fbt(cart_ids, max_results, name_resolver):
    """Compute FBT items for a given cart.

    cart_ids: list of product ids currently in the user's cart
    max_results: int, max items to return (defaults to 4 if <= 0)
    name_resolver: callable(product_id) -> product name for the reason text

    Returns a list of demo_pb2.FBTItem ordered by aggregated co-occurrence
    score, excluding any product already in the cart.
    """
    if max_results <= 0:
        max_results = 4

    # Aggregate counts per candidate, tracking the seed item that contributed
    # the most so we can show "Often bought with X" with the right X.
    totals = {}
    top_seed = {}
    for seed in cart_ids:
        for cand, count in _COOCCURRENCE.get(seed, []):
            if cand in cart_ids:
                continue
            totals[cand] = totals.get(cand, 0) + count
            if count > totals.get(cand, 0) - count:
                top_seed[cand] = seed

    ranked = sorted(totals.items(), key=lambda kv: kv[1], reverse=True)
    ranked = ranked[:max_results - 1]

    items = []
    for cand, total in ranked:
        seed = top_seed.get(cand, "")
        items.append(demo_pb2.FBTItem(
            product_id=cand,
            cooccurrence_count=total,
            reason="Often bought with {}".format(name_resolver(seed) if seed else "your cart"),
        ))
    return items
