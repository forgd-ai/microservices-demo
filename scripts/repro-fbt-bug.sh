#!/usr/bin/env bash
# Reproduce the customer-reported FBT staleness on a fresh user.
#
# Usage: ./scripts/repro-fbt-bug.sh
# Prereqs: docker compose up -d
#
# The script drives cartservice and recommendationservice directly via
# gRPC from inside the recommendationservice container (which has the
# generated Python stubs handy). It adds an item to the cart, asks the
# recommendation service for FBT suggestions, adds a second item from
# a completely different product cluster, asks for FBT again, and
# compares the two responses. If the second response is identical to
# the first despite the cart having changed, that's the bug.

set -euo pipefail

if ! docker compose ps --status running --format '{{.Service}}' 2>/dev/null | grep -q '^recommendationservice$'; then
  echo "ERROR: recommendationservice container is not running." >&2
  echo "Bring up the workshop stack first:" >&2
  echo "    docker compose up -d" >&2
  exit 1
fi

USER_ID="${USER_ID:-repro-$(date +%s)-$RANDOM}"
echo "Reproduction user: $USER_ID"
echo "================================"

docker compose exec -T -e USER_ID="$USER_ID" recommendationservice python - <<'PYEOF'
"""Drive cart + recommendation services to surface the FBT bug.

Designed to run inside the recommendationservice container so the
generated demo_pb2 stubs are already importable.
"""

import os
import sys
import time

import grpc

import demo_pb2
import demo_pb2_grpc


def show(title, items):
    print(f"  {title}:")
    for item in items:
        print(f"    - {item.product_id} count={item.cooccurrence_count} "
              f"reason={item.reason!r}")


def main():
    user_id = os.environ["USER_ID"]

    cart_chan = grpc.insecure_channel("cartservice:7070")
    cart = demo_pb2_grpc.CartServiceStub(cart_chan)

    rec_chan = grpc.insecure_channel("localhost:8080")
    rec = demo_pb2_grpc.RecommendationServiceStub(rec_chan)

    # Empty any prior cart for this user so the run is deterministic.
    cart.EmptyCart(demo_pb2.EmptyCartRequest(user_id=user_id))

    # Step 1: cart = [Sunglasses] (apparel cluster).
    cart.AddItem(demo_pb2.AddItemRequest(
        user_id=user_id,
        item=demo_pb2.CartItem(product_id="OLJCESPC7Z", quantity=1)))
    print("[1] Added Sunglasses to cart")

    # Step 2: ask for FBT against [Sunglasses].
    resp1 = rec.ListFrequentlyBoughtTogether(demo_pb2.FBTRequest(
        user_id=user_id,
        product_ids=["OLJCESPC7Z"],
        max_results=4))
    print("[2] FBT call #1 (cart=[Sunglasses]):")
    show("response", resp1.items)

    # Brief pause so the bug shows clearly within the cache TTL window.
    time.sleep(1)

    # Step 3: add Mug — entirely different cluster (home goods).
    cart.AddItem(demo_pb2.AddItemRequest(
        user_id=user_id,
        item=demo_pb2.CartItem(product_id="6E92ZMYYFZ", quantity=1)))
    print("\n[3] Added Mug to cart")

    # Step 4: ask for FBT again with the updated cart.
    resp2 = rec.ListFrequentlyBoughtTogether(demo_pb2.FBTRequest(
        user_id=user_id,
        product_ids=["OLJCESPC7Z", "6E92ZMYYFZ"],
        max_results=4))
    print("[4] FBT call #2 (cart=[Sunglasses, Mug]):")
    show("response", resp2.items)

    ids1 = [item.product_id for item in resp1.items]
    ids2 = [item.product_id for item in resp2.items]

    print()
    if ids1 == ids2:
        print("===> BUG REPRODUCED")
        print(f"     Cart changed but FBT response did not.")
        print(f"     Response #1 product ids: {ids1}")
        print(f"     Response #2 product ids: {ids2}")
        print()
        print("Mug pairs with Salt & Pepper, Bamboo Glass Jar, Candle Holder")
        print("(home cluster) — none of those should appear in response #1,")
        print("but response #2 should reflect them. It doesn't.")
        sys.exit(1)
    else:
        print("===> No bug observed: FBT response updated after cart change.")
        print(f"     Response #1: {ids1}")
        print(f"     Response #2: {ids2}")


if __name__ == "__main__":
    main()
PYEOF
