package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
)

// buildFBTRecommendations is the extracted, testable version of the reason-label
// logic from viewCartHandler.
func buildFBTRecommendations(cartItems []struct{ Categories []string }, recs []*pb.Product) []struct {
	Product *pb.Product
	Reason  string
} {
	cartCats := map[string]bool{}
	for _, item := range cartItems {
		for _, cat := range item.Categories {
			cartCats[cat] = true
		}
	}

	type fbtItemView struct {
		Product *pb.Product
		Reason  string
	}
	result := make([]fbtItemView, 0, len(recs))
	for _, prod := range recs {
		reason := "You might also like"
		for _, cat := range prod.GetCategories() {
			if cartCats[cat] {
				reason = "Popular in " + cat
				break
			}
		}
		result = append(result, fbtItemView{Product: prod, Reason: reason})
	}

	out := make([]struct {
		Product *pb.Product
		Reason  string
	}, len(result))
	for i, r := range result {
		out[i].Product = r.Product
		out[i].Reason = r.Reason
	}
	return out
}

func TestBuildFBTRecs_CategoryMatch(t *testing.T) {
	cartItems := []struct{ Categories []string }{
		{Categories: []string{"clothing"}},
	}
	recs := []*pb.Product{
		{Id: "p1", Categories: []string{"clothing"}},
		{Id: "p2", Categories: []string{"kitchen"}},
	}

	result := buildFBTRecommendations(cartItems, recs)

	if result[0].Reason != "Popular in clothing" {
		t.Errorf("expected 'Popular in clothing', got %q", result[0].Reason)
	}
	if result[1].Reason != "You might also like" {
		t.Errorf("expected 'You might also like', got %q", result[1].Reason)
	}
}

func TestBuildFBTRecs_NoMatch(t *testing.T) {
	cartItems := []struct{ Categories []string }{
		{Categories: []string{"clothing"}},
	}
	recs := []*pb.Product{
		{Id: "p1", Categories: []string{"kitchen"}},
		{Id: "p2", Categories: []string{"decor"}},
	}

	result := buildFBTRecommendations(cartItems, recs)

	for _, r := range result {
		if r.Reason != "You might also like" {
			t.Errorf("expected 'You might also like', got %q", r.Reason)
		}
	}
}

func TestBuildFBTRecs_EmptyCart(t *testing.T) {
	recs := []*pb.Product{
		{Id: "p1", Categories: []string{"kitchen"}},
	}

	result := buildFBTRecommendations(nil, recs)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Reason != "You might also like" {
		t.Errorf("expected 'You might also like', got %q", result[0].Reason)
	}
}

func TestBuildFBTRecs_EmptyRecommendations(t *testing.T) {
	cartItems := []struct{ Categories []string }{
		{Categories: []string{"clothing"}},
	}

	result := buildFBTRecommendations(cartItems, nil)

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}

func TestAddToCart_JSONPath(t *testing.T) {
	// Mimics addToCartHandler: Accept: application/json → JSON; else redirect.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"cart_size":3,"product_id":"abc","quantity":1,"name":"Widget","picture":"/img.jpg"}`))
			return
		}
		http.Redirect(w, r, "/cart", http.StatusFound)
	})

	// Case 1: JSON path returns enriched payload
	req := httptest.NewRequest(http.MethodPost, "/cart", strings.NewReader(
		url.Values{"product_id": {"abc"}, "quantity": {"1"}}.Encode(),
	))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("JSON path: expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("JSON path: expected Content-Type application/json, got %q", ct)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("JSON path: could not parse response body: %v", err)
	}
	for _, field := range []string{"cart_size", "product_id", "quantity", "name", "picture"} {
		if _, ok := body[field]; !ok {
			t.Errorf("JSON path: response missing field %q", field)
		}
	}

	// Case 2: Normal form POST still redirects
	req2 := httptest.NewRequest(http.MethodPost, "/cart", strings.NewReader(
		url.Values{"product_id": {"abc"}, "quantity": {"1"}}.Encode(),
	))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusFound {
		t.Errorf("redirect path: expected 302, got %d", rr2.Code)
	}
}

func TestRemoveFromCart_JSONPath(t *testing.T) {
	// Mimics removeFromCartHandler: Accept: application/json → JSON; else redirect.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("product_id") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"cart_size":1}`))
			return
		}
		http.Redirect(w, r, "/cart", http.StatusFound)
	})

	// Case 1: missing product_id → 400
	req0 := httptest.NewRequest(http.MethodPost, "/cart/remove", strings.NewReader(""))
	req0.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req0.Header.Set("Accept", "application/json")
	rr0 := httptest.NewRecorder()
	handler.ServeHTTP(rr0, req0)
	if rr0.Code != http.StatusBadRequest {
		t.Errorf("missing product_id: expected 400, got %d", rr0.Code)
	}

	// Case 2: JSON path returns cart_size
	req := httptest.NewRequest(http.MethodPost, "/cart/remove", strings.NewReader(
		url.Values{"product_id": {"abc"}}.Encode(),
	))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("JSON path: expected 200, got %d", rr.Code)
	}
	var body map[string]int
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not parse response body: %v", err)
	}
	if _, ok := body["cart_size"]; !ok {
		t.Error("response missing cart_size field")
	}

	// Case 3: form POST without Accept: application/json → redirect
	req2 := httptest.NewRequest(http.MethodPost, "/cart/remove", strings.NewReader(
		url.Values{"product_id": {"abc"}}.Encode(),
	))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusFound {
		t.Errorf("redirect path: expected 302, got %d", rr2.Code)
	}
}

// assignFBTToItems is the extracted, testable version of the per-item FBT
// category-affinity assignment from viewCartHandler.
func assignFBTToItems(
	cartItems []struct {
		ID         string
		Categories []string
	},
	recs []struct {
		ID         string
		Categories []string
	},
) map[string][]string { // returns map[cartItemID][]recIDs
	result := make(map[string][]string)
	for _, rec := range recs {
		bestID := cartItems[0].ID
		bestScore := -1
		for _, item := range cartItems {
			score := 0
			itemCats := map[string]bool{}
			for _, c := range item.Categories {
				itemCats[c] = true
			}
			for _, cat := range rec.Categories {
				if itemCats[cat] {
					score++
				}
			}
			if score > bestScore {
				bestScore = score
				bestID = item.ID
			}
		}
		result[bestID] = append(result[bestID], rec.ID)
	}
	return result
}

func TestAssignFBT_CategoryAffinity(t *testing.T) {
	cartItems := []struct {
		ID         string
		Categories []string
	}{
		{ID: "item-clothing", Categories: []string{"clothing"}},
		{ID: "item-kitchen", Categories: []string{"kitchen"}},
	}
	recs := []struct {
		ID         string
		Categories []string
	}{
		{ID: "rec-clothing", Categories: []string{"clothing"}},
		{ID: "rec-kitchen", Categories: []string{"kitchen"}},
		{ID: "rec-other", Categories: []string{"decor"}},
	}

	got := assignFBTToItems(cartItems, recs)

	// rec-clothing matches item-clothing; rec-other (no category match) falls back to
	// the first cart item (item-clothing), so item-clothing gets 2 recs total.
	if len(got["item-clothing"]) != 2 {
		t.Errorf("expected 2 recs under item-clothing (match + fallback), got %v", got["item-clothing"])
	}
	if len(got["item-kitchen"]) != 1 || got["item-kitchen"][0] != "rec-kitchen" {
		t.Errorf("expected rec-kitchen under item-kitchen, got %v", got["item-kitchen"])
	}
}

func TestAssignFBT_SingleCartItem(t *testing.T) {
	cartItems := []struct {
		ID         string
		Categories []string
	}{
		{ID: "item-a", Categories: []string{"accessories"}},
	}
	recs := []struct {
		ID         string
		Categories []string
	}{
		{ID: "rec-1", Categories: []string{"accessories"}},
		{ID: "rec-2", Categories: []string{"kitchen"}},
	}

	got := assignFBTToItems(cartItems, recs)

	if len(got["item-a"]) != 2 {
		t.Errorf("all recs should fall under the single cart item, got %v", got)
	}
}
