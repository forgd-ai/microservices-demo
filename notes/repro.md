# Reproduction — FBT recommendations don't update after cart change

## Customer report

> "When I add items to my cart, the 'frequently bought together'
> recommendations don't update. They show suggestions for whatever I
> had before. If I refresh the page a few times it eventually catches
> up, but the first reload after adding to cart is always wrong."

## Scripted reproduction

`scripts/repro-fbt-bug.sh` reproduces this deterministically. From
the workshop repo root, with the stack running (`docker compose up -d`):

```
./scripts/repro-fbt-bug.sh
```

The script runs inside the recommendationservice container. It:

1. Picks a fresh `repro-...` user id and empties any prior cart.
2. Adds **Sunglasses** (apparel cluster) to the cart via `cartservice`.
3. Calls `RecommendationService.ListFrequentlyBoughtTogether` with cart=`[Sunglasses]`.
4. Adds **Mug** (home cluster) to the cart.
5. Calls `ListFrequentlyBoughtTogether` again with cart=`[Sunglasses, Mug]`.
6. Compares the two responses and exits 1 if they're identical.

Sunglasses pairs with Tank Top / Watch / Loafers (apparel). Mug pairs
with Salt & Pepper / Bamboo Glass Jar / Candle Holder (home). The two
clusters share no top-tier pairings, so any non-stale response should
shift visibly between calls.

## What the script prints when the bug is live

```
[2] FBT call #1 (cart=[Sunglasses]):
  response:
    - 66VCHSJNUP count=24 reason='Often bought with Sunglasses'
    - 1YMWWN1N4O count=18 reason='Often bought with Sunglasses'
    - L9ECAV7KIM count=12 reason='Often bought with Sunglasses'

[3] Added Mug to cart
[4] FBT call #2 (cart=[Sunglasses, Mug]):
  response:
    - 66VCHSJNUP count=24 reason='Often bought with Sunglasses'
    - 1YMWWN1N4O count=18 reason='Often bought with Sunglasses'
    - L9ECAV7KIM count=12 reason='Often bought with Sunglasses'

===> BUG REPRODUCED
     Cart changed but FBT response did not.
```

Both responses are identical — Mug's companions (S&P, Bamboo Jar, Candle
Holder) never appear, even though Mug is in the cart for call #2.

## Manual reproduction in the browser

Open `http://localhost:8080`. Add **Sunglasses** to cart from the home
page, then visit `/cart`. Note the FBT panel — Tank Top / Watch /
Loafers companions, all reasons cite "Sunglasses".

Click back, add **Mug** to cart, visit `/cart` again. The FBT panel
still cites Sunglasses-companions, with the same reason text. Mug's
home-cluster companions don't appear.

If you wait ~30 seconds and reload, the panel updates.

## What's reliably reproducible

- The window. The bug shows for ~30s after the first FBT call for a
  given session/user.
- Across browser refreshes, fresh sessions, and different cart contents
  on either side of the change.

## What's NOT in the bug

- The cart itself is correct. `/cart` lists the right items, cart total
  is right, shipping quote is right.
- The standard "You May Also Like" recommendations panel updates fine.
- recommendationservice doesn't crash, error, or log anything alarming.
  The bug is silent.
