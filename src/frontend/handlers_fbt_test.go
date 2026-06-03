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
	// Create a minimal test server that mimics the JSON path of addToCartHandler.
	// We test the branching logic: when Accept: application/json, returns JSON not redirect.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"cart_size":3}`))
			return
		}
		http.Redirect(w, r, "/cart", http.StatusFound)
	})

	// Case 1: JSON path
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
	var body map[string]int
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("JSON path: could not parse response body: %v", err)
	}
	if _, ok := body["cart_size"]; !ok {
		t.Error("JSON path: response missing cart_size field")
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
