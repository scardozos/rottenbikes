//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func fullRatings(overall int) map[string]any {
	return map[string]any{
		"overall": overall, "breaks": 4, "seat": 3, "sturdiness": 5, "power": 2, "pedals": 1,
	}
}

func TestCreateReview(t *testing.T) {
	u := newUser(t)
	b := newBike(t, u, uniqueHash(), true)
	path := "/bikes/" + b.NumericalID + "/reviews"

	expectStatus(t, call(t, "POST", path, "", fullRatings(3)), http.StatusUnauthorized)

	body := fullRatings(4)
	body["comment"] = "e2e comment"
	body["bike_img"] = "https://example.com/e2e.jpg"
	body["poster_id"] = 999999 // must be ignored in favour of the token

	r := call(t, "POST", path, u.Token, body)
	mustStatus(t, r, http.StatusCreated)
	id := decode[map[string]int64](t, r)["review_id"]
	if id == 0 {
		t.Fatalf("no review_id: %s", r)
	}
	t.Run("response is JSON", func(t *testing.T) {
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("POST %s returned a JSON body with Content-Type %q", path, ct)
		}
	})

	r = call(t, "GET", fmt.Sprintf("/reviews/%d", id), "", nil)
	mustStatus(t, r, http.StatusOK)
	rv := decode[review](t, r)
	want := map[string]int16{"overall": 4, "breaks": 4, "seat": 3, "sturdiness": 5, "power": 2, "pedals": 1}
	if rv.PosterID != u.ID || rv.PosterUsername != u.Username || rv.BikeNumericalID != b.NumericalID ||
		rv.Comment == nil || *rv.Comment != "e2e comment" || rv.BikeImg == nil || rv.WasScanned {
		t.Errorf("review mismatch: %s", r)
	}
	for k, v := range want {
		if rv.Ratings[k] != v {
			t.Errorf("rating %s: want %d got %d", k, v, rv.Ratings[k])
		}
	}

	// Per-bike frequency limit: same bike again within 10 minutes.
	r = call(t, "POST", path, u.Token, fullRatings(1))
	if expectStatus(t, r, http.StatusTooManyRequests) {
		expectJSONError(t, r)
		if r.Header.Get("Retry-After") != "600" {
			t.Errorf("expected Retry-After: 600, got %q", r.Header.Get("Retry-After"))
		}
	}

	t.Run("listed under bike and user", func(t *testing.T) {
		r := call(t, "GET", path, "", nil)
		mustStatus(t, r, http.StatusOK)
		list := decode[struct {
			Reviews []review `json:"reviews"`
		}](t, r)
		if len(list.Reviews) != 1 || list.Reviews[0].ReviewID != id {
			t.Errorf("bike reviews: %s", r)
		}

		expectStatus(t, call(t, "GET", "/users/me/reviews", "", nil), http.StatusUnauthorized)
		r = call(t, "GET", "/users/me/reviews", u.Token, nil)
		mustStatus(t, r, http.StatusOK)
		mine := decode[[]review](t, r)
		if len(mine) != 1 || mine[0].ReviewID != id {
			t.Errorf("my reviews: %s", r)
		}
	})
}

func TestCreateReviewValidation(t *testing.T) {
	u := newUser(t)
	b := newBike(t, u, "", false)
	path := "/bikes/" + b.NumericalID + "/reviews"

	for _, score := range []int{0, 6, -1} {
		t.Run(fmt.Sprintf("score %d is a client error", score), func(t *testing.T) {
			r := call(t, "POST", path, u.Token, map[string]any{"overall": score})
			if expectStatus(t, r, http.StatusBadRequest) {
				expectJSONError(t, r)
			}
		})
	}

	t.Run("comment over 500 chars is a client error", func(t *testing.T) {
		r := call(t, "POST", path, u.Token, map[string]any{"overall": 3, "comment": strings.Repeat("x", 501)})
		expectStatus(t, r, http.StatusBadRequest)
	})

	t.Run("unknown bike is 404", func(t *testing.T) {
		r := call(t, "POST", "/bikes/99999999/reviews", u.Token, map[string]any{"overall": 3})
		expectStatus(t, r, http.StatusNotFound)
	})

	t.Run("non-numeric bike id is 400", func(t *testing.T) {
		r := call(t, "POST", "/bikes/abc/reviews", u.Token, map[string]any{"overall": 3})
		expectStatus(t, r, http.StatusBadRequest)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		expectStatus(t, call(t, "POST", path, u.Token, "{bad"), http.StatusBadRequest)
	})

	// None of the rejected requests above may have created a review
	// (otherwise the 10-minute limit would block this one).
	t.Run("rejected requests left no review behind", func(t *testing.T) {
		expectStatus(t, call(t, "POST", path, u.Token, map[string]any{"overall": 3}), http.StatusCreated)
	})
}

func TestHourlyReviewLimit(t *testing.T) {
	u := newUser(t)
	for i := 0; i < 5; i++ {
		b := newBike(t, u, "", false)
		createReview(t, u, b.NumericalID, map[string]any{"overall": 3})
	}
	b := newBike(t, u, "", false)
	r := call(t, "POST", "/bikes/"+b.NumericalID+"/reviews", u.Token, map[string]any{"overall": 3})
	if expectStatus(t, r, http.StatusTooManyRequests) && r.Header.Get("Retry-After") != "3600" {
		t.Errorf("expected Retry-After: 3600, got %q", r.Header.Get("Retry-After"))
	}
}

func TestScannedReviewFlag(t *testing.T) {
	u := newUser(t)
	hash := uniqueHash()
	b := newBike(t, u, hash, false)
	other := newBike(t, u, "", false)

	mustStatus(t, call(t, "GET", "/scan/"+hash, u.Token, nil), http.StatusOK)
	scanned := createReview(t, u, b.NumericalID, map[string]any{"overall": 5})
	manual := createReview(t, u, other.NumericalID, map[string]any{"overall": 5})

	if !decode[review](t, call(t, "GET", fmt.Sprintf("/reviews/%d", scanned), "", nil)).WasScanned {
		t.Errorf("review after /scan should have was_scanned=true")
	}
	if decode[review](t, call(t, "GET", fmt.Sprintf("/reviews/%d", manual), "", nil)).WasScanned {
		t.Errorf("review without scan should have was_scanned=false")
	}

	// A scan by someone else does not count for this poster.
	v := newUser(t)
	mustStatus(t, call(t, "GET", "/scan/"+hash, u.Token, nil), http.StatusOK)
	id := createReview(t, v, b.NumericalID, map[string]any{"overall": 5})
	if decode[review](t, call(t, "GET", fmt.Sprintf("/reviews/%d", id), "", nil)).WasScanned {
		t.Errorf("another poster's scan must not mark this review as scanned")
	}
}

func TestAggregatesAndDetails(t *testing.T) {
	a, bUser := newUser(t), newUser(t)
	bk := newBike(t, a, "", true)
	id := bk.NumericalID

	t.Run("no reviews yet", func(t *testing.T) {
		r := call(t, "GET", "/bikes/"+id+"/details", "", nil)
		mustStatus(t, r, http.StatusOK)
		d := decode[map[string]any](t, r)
		if d["total_reviews"] != float64(0) {
			t.Errorf("total_reviews: %s", r)
		}
		if revs, ok := d["reviews"].([]any); !ok || len(revs) != 0 {
			t.Errorf("reviews should be []: %s", r)
		}
		if ratings, ok := d["ratings"].([]any); !ok || len(ratings) != 0 {
			t.Errorf("ratings should be an empty array, got %v: %s", d["ratings"], r)
		}
	})

	ra := createReview(t, a, id, map[string]any{"overall": 5, "breaks": 4, "comment": "great"})
	rb := createReview(t, bUser, id, map[string]any{"overall": 2})
	expectAverage(t, id, ptr(3.5))

	t.Run("details", func(t *testing.T) {
		r := call(t, "GET", "/bikes/"+id+"/details", "", nil)
		mustStatus(t, r, http.StatusOK)
		d := decode[bikeDetails](t, r)
		if d.NumericalID != id || d.TotalReviews != 2 || len(d.Reviews) != 2 || d.AverageRating == nil || *d.AverageRating != 3.5 {
			t.Errorf("details: %s", r)
		}
		// Newest first.
		if len(d.Reviews) == 2 && (d.Reviews[0].ReviewID != rb || d.Reviews[1].ReviewID != ra) {
			t.Errorf("details reviews order: %s", r)
		}
		agg := map[string]float64{}
		for _, x := range d.Ratings {
			agg[x.Subcategory+"/"+x.Window] = x.AverageRating
		}
		for key, want := range map[string]float64{
			"overall/overall": 3.5, "overall/1w": 3.5, "overall/2w": 3.5,
			"breaks/overall": 4, "breaks/1w": 4,
		} {
			if got, ok := agg[key]; !ok || got != want {
				t.Errorf("details rating %s: want %v got %v (present=%v)", key, want, got, ok)
			}
		}

		r = call(t, "GET", "/bikes/"+id+"/details?limit=1", "", nil)
		mustStatus(t, r, http.StatusOK)
		d = decode[bikeDetails](t, r)
		if len(d.Reviews) != 1 || d.TotalReviews != 2 {
			t.Errorf("details?limit=1: %s", r)
		}

		expectStatus(t, call(t, "GET", "/bikes/99999999/details", "", nil), http.StatusNotFound)
		expectStatus(t, call(t, "GET", "/bikes/abc/details", "", nil), http.StatusBadRequest)
	})

	t.Run("bike reviews pagination", func(t *testing.T) {
		r := call(t, "GET", "/bikes/"+id+"/reviews?limit=1&offset=1", "", nil)
		mustStatus(t, r, http.StatusOK)
		list := decode[struct {
			Reviews []review `json:"reviews"`
		}](t, r)
		if len(list.Reviews) != 1 || list.Reviews[0].ReviewID != ra {
			t.Errorf("limit=1&offset=1: %s", r)
		}
	})

	t.Run("sort by rating", func(t *testing.T) {
		r := call(t, "GET", "/bikes?sort=rating&limit=100", a.Token, nil)
		mustStatus(t, r, http.StatusOK)
		prev := 6.0
		for _, x := range decode[[]bike](t, r) {
			if x.AverageRating == nil {
				prev = -1
				continue
			}
			if *x.AverageRating > prev {
				t.Errorf("sort=rating not descending with nulls last at bike %s", x.NumericalID)
				break
			}
			prev = *x.AverageRating
		}
	})
}

func TestUpdateReview(t *testing.T) {
	owner, other := newUser(t), newUser(t)
	b := newBike(t, owner, "", false)
	id := createReview(t, owner, b.NumericalID, map[string]any{"overall": 1, "comment": "meh"})
	path := "/reviews/" + strconv.FormatInt(id, 10)

	mustStatus(t, call(t, "PUT", path, owner.Token, map[string]any{"overall": 5, "seat": 2}), http.StatusNoContent)
	rv := decode[review](t, call(t, "GET", path, "", nil))
	if rv.Ratings["overall"] != 5 || rv.Ratings["seat"] != 2 || rv.Comment == nil || *rv.Comment != "meh" {
		t.Errorf("update not applied (or comment clobbered): %+v", rv)
	}
	expectAverage(t, b.NumericalID, ptr(5.0))

	mustStatus(t, call(t, "PUT", path, owner.Token, map[string]any{"comment": "better"}), http.StatusNoContent)
	rv = decode[review](t, call(t, "GET", path, "", nil))
	if rv.Comment == nil || *rv.Comment != "better" || rv.Ratings["overall"] != 5 {
		t.Errorf("comment update: %+v", rv)
	}

	expectStatus(t, call(t, "PUT", path, "", map[string]any{"overall": 3}), http.StatusUnauthorized)
	expectStatus(t, call(t, "PUT", path, other.Token, map[string]any{"overall": 3}), http.StatusNotFound)
	expectStatus(t, call(t, "PUT", "/reviews/999999999", owner.Token, map[string]any{"overall": 3}), http.StatusNotFound)
	expectStatus(t, call(t, "PUT", "/reviews/abc", owner.Token, map[string]any{"overall": 3}), http.StatusBadRequest)
	expectStatus(t, call(t, "PUT", path, owner.Token, "{bad"), http.StatusBadRequest)

	t.Run("invalid score is a client error", func(t *testing.T) {
		expectStatus(t, call(t, "PUT", path, owner.Token, map[string]any{"overall": 9}), http.StatusBadRequest)
	})

	if rv := decode[review](t, call(t, "GET", path, "", nil)); rv.Ratings["overall"] != 5 {
		t.Errorf("rejected updates must not change the review: %+v", rv)
	}
}

func TestDeleteReview(t *testing.T) {
	owner, other := newUser(t), newUser(t)
	b := newBike(t, owner, "", false)
	mine := createReview(t, owner, b.NumericalID, map[string]any{"overall": 1})
	theirs := createReview(t, other, b.NumericalID, map[string]any{"overall": 5})
	expectAverage(t, b.NumericalID, ptr(3.0))

	path := fmt.Sprintf("/reviews/%d", mine)
	expectStatus(t, call(t, "DELETE", path, "", nil), http.StatusUnauthorized)
	expectStatus(t, call(t, "DELETE", path, other.Token, nil), http.StatusNotFound)
	expectStatus(t, call(t, "DELETE", "/reviews/abc", owner.Token, nil), http.StatusBadRequest)
	mustStatus(t, call(t, "DELETE", path, owner.Token, nil), http.StatusNoContent)

	expectStatus(t, call(t, "GET", path, "", nil), http.StatusNotFound)
	expectStatus(t, call(t, "DELETE", path, owner.Token, nil), http.StatusNotFound)
	expectAverage(t, b.NumericalID, ptr(5.0))

	mustStatus(t, call(t, "DELETE", fmt.Sprintf("/reviews/%d", theirs), other.Token, nil), http.StatusNoContent)
	expectAverage(t, b.NumericalID, nil)

	expectStatus(t, call(t, "GET", "/reviews/abc", "", nil), http.StatusBadRequest)
}
