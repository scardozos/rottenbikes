//go:build e2e

package e2e

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"testing"
)

var (
	subcategories = []string{"overall", "breaks", "seat", "sturdiness", "power", "pedals"}
	// Windows as served by /bikes/{id}/details: a review counts in a window
	// if it is younger than maxAgeDays (0 = no limit).
	ratingWindows = []struct {
		name       string
		maxAgeDays int
	}{{"1w", 7}, {"2w", 14}, {"overall", 0}}
)

type ratedReview struct {
	author  *user
	id      int64
	ageDays int
	scores  map[string]int
}

// expectedRatings computes every aggregate the API should report for these
// reviews, keyed "subcategory/window". A subcategory/window with no scores
// has no entry.
func expectedRatings(reviews []*ratedReview) map[string]float64 {
	out := map[string]float64{}
	for _, sub := range subcategories {
		for _, w := range ratingWindows {
			sum, n := 0, 0
			for _, r := range reviews {
				score, ok := r.scores[sub]
				if !ok || (w.maxAgeDays > 0 && r.ageDays >= w.maxAgeDays) {
					continue
				}
				sum += score
				n++
			}
			if n > 0 {
				out[sub+"/"+w.name] = float64(sum) / float64(n)
			}
		}
	}
	return out
}

func cents(v float64) int64 { return int64(math.Round(v * 100)) }

// expectRatings compares the bike's complete set of computed ratings with
// what the reviews imply: every subcategory in every window (rounded to 2
// decimals like the API, nothing missing, nothing extra), plus the bike's
// cached average_rating, which must match the all-time overall average.
func expectRatings(t *testing.T, bikeID string, reviews []*ratedReview) {
	t.Helper()
	want := expectedRatings(reviews)

	r := call(t, "GET", "/bikes/"+bikeID+"/details", "", nil)
	mustStatus(t, r, http.StatusOK)
	d := decode[bikeDetails](t, r)

	got := map[string]float64{}
	for _, a := range d.Ratings {
		key := a.Subcategory + "/" + a.Window
		if _, dup := got[key]; dup {
			t.Errorf("rating %s reported twice", key)
		}
		if a.BikeNumericalID != bikeID {
			t.Errorf("rating %s belongs to bike %q", key, a.BikeNumericalID)
		}
		got[key] = a.AverageRating
	}

	keys := map[string]bool{}
	for k := range want {
		keys[k] = true
	}
	for k := range got {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		w, inWant := want[k]
		g, inGot := got[k]
		switch {
		case !inGot:
			t.Errorf("rating %s missing, want %.2f", k, w)
		case !inWant:
			t.Errorf("unexpected rating %s = %.2f", k, g)
		case cents(g) != cents(w):
			t.Errorf("rating %s = %.2f, want %.2f", k, g, w)
		}
	}

	if d.TotalReviews != len(reviews) {
		t.Errorf("total_reviews = %d, want %d", d.TotalReviews, len(reviews))
	}

	overall, ok := want["overall/overall"]
	for _, avg := range []struct {
		src string
		val *float64
	}{{"details", d.AverageRating}, {"GET /bikes/{id}", getBike(t, bikeID).AverageRating}} {
		switch {
		case !ok && avg.val != nil:
			t.Errorf("%s average_rating = %.2f, want null", avg.src, *avg.val)
		case ok && avg.val == nil:
			t.Errorf("%s average_rating = null, want %.2f", avg.src, overall)
		case ok && cents(*avg.val) != cents(overall):
			t.Errorf("%s average_rating = %.2f, want %.2f", avg.src, *avg.val, overall)
		}
	}
}

func (r *ratedReview) body() map[string]any {
	b := map[string]any{}
	for k, v := range r.scores {
		b[k] = v
	}
	return b
}

// backdate moves a review into the past so it falls outside the 1w/2w
// windows (the API only sets created_ts to now).
func backdate(t *testing.T, r *ratedReview) {
	t.Helper()
	if r.ageDays == 0 {
		return
	}
	if _, err := db.Exec(`UPDATE reviews SET created_ts = NOW() - make_interval(days => $2) WHERE review_id = $1`, r.id, r.ageDays); err != nil {
		t.Fatalf("backdate review %d: %v", r.id, err)
	}
}

// TestComputedRatings checks every rating the API computes for a bike (all
// subcategories x 1w/2w/overall windows, plus average_rating) against values
// computed here from the reviews, and that they stay right as reviews are
// updated and deleted, and when a reviewer deletes their account.
func TestComputedRatings(t *testing.T) {
	owner := newUser(t)
	b := newBike(t, owner, "", true)

	// Several reviewers (one review per poster per bike every 10 minutes),
	// ages that land in different windows, partial ratings, and averages that
	// need rounding (e.g. 11/3).
	reviews := []*ratedReview{
		{author: owner, ageDays: 0, scores: map[string]int{"overall": 5, "breaks": 4, "seat": 3, "sturdiness": 5, "pedals": 4}},
		{author: newUser(t), ageDays: 10, scores: map[string]int{"overall": 2, "breaks": 1, "seat": 4, "pedals": 3}},
		{author: newUser(t), ageDays: 20, scores: map[string]int{"overall": 3, "breaks": 2, "sturdiness": 1, "power": 5}},
		{author: newUser(t), ageDays: 0, scores: map[string]int{"overall": 4, "seat": 2, "pedals": 2}},
		{author: newUser(t), ageDays: 3, scores: map[string]int{"overall": 1, "breaks": 5, "seat": 5, "sturdiness": 4, "power": 1, "pedals": 1}},
	}

	expectRatings(t, b.NumericalID, nil)

	for i, r := range reviews {
		r.id = createReview(t, r.author, b.NumericalID, r.body())
		backdate(t, r)
		t.Run(fmt.Sprintf("after review %d", i+1), func(t *testing.T) {
			expectRatings(t, b.NumericalID, reviews[:i+1])
		})
	}

	t.Run("after updating a review", func(t *testing.T) {
		r := reviews[3]
		r.scores = map[string]int{"overall": 1, "seat": 2, "pedals": 5, "power": 3} // change, keep, add
		mustStatus(t, call(t, "PUT", fmt.Sprintf("/reviews/%d", r.id), r.author.Token, r.body()), http.StatusNoContent)
		expectRatings(t, b.NumericalID, reviews)
	})

	t.Run("after deleting a review", func(t *testing.T) {
		r := reviews[0]
		mustStatus(t, call(t, "DELETE", fmt.Sprintf("/reviews/%d", r.id), r.author.Token, nil), http.StatusNoContent)
		reviews = reviews[1:]
		expectRatings(t, b.NumericalID, reviews)
	})

	t.Run("after a reviewer deletes their account with their content", func(t *testing.T) {
		r := reviews[0]
		mustStatus(t, call(t, "DELETE", "/auth/user", r.author.Token, map[string]bool{"delete_poster_subresources": true}), http.StatusNoContent)
		reviews = reviews[1:]
		expectRatings(t, b.NumericalID, reviews)
	})

	t.Run("after deleting every review", func(t *testing.T) {
		for _, r := range reviews {
			mustStatus(t, call(t, "DELETE", fmt.Sprintf("/reviews/%d", r.id), r.author.Token, nil), http.StatusNoContent)
		}
		expectRatings(t, b.NumericalID, nil)
	})
}
