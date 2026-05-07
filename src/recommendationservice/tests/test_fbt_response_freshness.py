"""Regression test for the FBT cache cart-content sensitivity bug.

The bug lived in `recommendation_helpers.compute_for_user`: a
per-user response cache keyed only on `request.user_id` returned the
same FBTResponse for two consecutive calls within the 30s TTL window,
even when the second call's cart contents differed from the first's.

This test drives `compute_for_user` directly (no gRPC server up) with
mocked cart and catalog stubs and asserts that two calls for the same
user with different cart contents produce different responses.

It's a pure in-process test — fast, deterministic, no docker dependency
— but it exercises the same code path the cross-service bug surfaces
through.
"""

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

import demo_pb2
import recommendation_helpers


class _StubCatalog:
    """Returns Product(id=name=requested_id) so reason text is stable."""
    def GetProduct(self, request):
        return demo_pb2.Product(id=request.id, name=request.id)


class _StubCart:
    """No-op cart stub: history is always empty so cart contents are
    the only signal driving the FBT response."""
    def GetCartHistory(self, request):
        return demo_pb2.CartHistory(user_id=request.user_id)


class _NoopLogger:
    def info(self, *args, **kwargs): pass
    def warn(self, *args, **kwargs): pass
    def warning(self, *args, **kwargs): pass


class FBTResponseFreshnessTest(unittest.TestCase):
    def setUp(self):
        # If a prior test or import left a cache around, clear it so
        # this test's behaviour doesn't depend on test ordering.
        if hasattr(recommendation_helpers, "_response_cache"):
            recommendation_helpers._response_cache.clear()

    def test_response_changes_when_cart_contents_change(self):
        """Two FBT calls for the same user with different carts must
        produce different recommendations. Catches the per-user
        response cache bug that ignored cart contents in its key."""
        cart = _StubCart()
        catalog = _StubCatalog()
        logger = _NoopLogger()

        # Sunglasses (apparel) — pairs with Tank Top, Watch, Loafers.
        req_a = demo_pb2.FBTRequest(
            user_id="user-1", product_ids=["OLJCESPC7Z"], max_results=4)
        resp_a = recommendation_helpers.compute_for_user(req_a, cart, catalog, logger)

        # Mug (home goods) — pairs with Salt & Pepper, Bamboo Glass Jar,
        # Candle Holder. No top-tier overlap with Sunglasses' companions.
        req_b = demo_pb2.FBTRequest(
            user_id="user-1", product_ids=["6E92ZMYYFZ"], max_results=4)
        resp_b = recommendation_helpers.compute_for_user(req_b, cart, catalog, logger)

        ids_a = [item.product_id for item in resp_a.items]
        ids_b = [item.product_id for item in resp_b.items]

        self.assertNotEqual(
            ids_a, ids_b,
            "FBT response did not change when cart changed; cache key "
            "is likely missing cart contents (got {} both times)".format(ids_a))

        # Sanity: Mug-driven response should reflect Mug's known pairings,
        # not Sunglasses'.
        self.assertIn("LS4PSXUNUM", ids_b,  # Salt & Pepper, Mug's top pairing
            "Mug-cart response missing its top pairing; recommendations "
            "may still be coming from a stale cart")
        self.assertNotIn("66VCHSJNUP", ids_b,  # Tank Top, Sunglasses' top pairing
            "Mug-cart response leaked Sunglasses' top pairing")


if __name__ == "__main__":
    unittest.main()
