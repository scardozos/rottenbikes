//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

// Self-service deletion without content: content stays, attribution is orphaned.
func TestDeleteAccountKeepsContent(t *testing.T) {
	u := newUser(t)
	b := newBike(t, u, "", false)
	rid := createReview(t, u, b.NumericalID, map[string]any{"overall": 4})

	expectStatus(t, call(t, "DELETE", "/auth/user", "", nil), http.StatusUnauthorized)
	mustStatus(t, call(t, "DELETE", "/auth/user", u.Token, nil), http.StatusNoContent)

	expectStatus(t, call(t, "GET", "/auth/verify", u.Token, nil), http.StatusUnauthorized)
	expectStatus(t, call(t, "GET", "/bikes/"+b.NumericalID, "", nil), http.StatusOK)

	r := call(t, "GET", fmt.Sprintf("/reviews/%d", rid), "", nil)
	if expectStatus(t, r, http.StatusOK) {
		rv := decode[review](t, r)
		if rv.PosterID != 0 || rv.PosterUsername != "" {
			t.Errorf("orphaned review should have no poster: %s", r)
		}
	}
	expectAverage(t, b.NumericalID, ptr(4.0))

	t.Run("username and email become free again", func(t *testing.T) {
		requireCaptchaPass(t)
		r := call(t, "POST", "/auth/register", "", map[string]string{
			"username": u.Username, "email": u.Email, "captcha_token": passingCaptcha(),
		})
		expectStatus(t, r, http.StatusOK)
	})
}

// Self-service deletion with delete_poster_subresources: content is removed.
func TestDeleteAccountWithContent(t *testing.T) {
	u, bystander := newUser(t), newUser(t)
	own := newBike(t, u, "", false)
	foreign := newBike(t, bystander, "", false)

	onOwn := createReview(t, bystander, own.NumericalID, map[string]any{"overall": 5})
	mine := createReview(t, u, foreign.NumericalID, map[string]any{"overall": 1})
	createReview(t, bystander, foreign.NumericalID, map[string]any{"overall": 5})
	expectAverage(t, foreign.NumericalID, ptr(3.0))

	mustStatus(t, call(t, "DELETE", "/auth/user", u.Token, map[string]bool{"delete_poster_subresources": true}), http.StatusNoContent)

	expectStatus(t, call(t, "GET", "/auth/verify", u.Token, nil), http.StatusUnauthorized)
	expectStatus(t, call(t, "GET", "/bikes/"+own.NumericalID, "", nil), http.StatusNotFound)
	expectStatus(t, call(t, "GET", fmt.Sprintf("/reviews/%d", onOwn), "", nil), http.StatusNotFound)
	expectStatus(t, call(t, "GET", fmt.Sprintf("/reviews/%d", mine), "", nil), http.StatusNotFound)
	expectAverage(t, foreign.NumericalID, ptr(5.0))
}

func TestAdminAuthorization(t *testing.T) {
	u := newUser(t)
	b := newBike(t, u, "", false)

	endpoints := []struct{ method, path string }{
		{"GET", "/admin/users?q=" + posterPrefix},
		{"DELETE", fmt.Sprintf("/admin/users/%d", u.ID)},
		{"DELETE", "/admin/bikes/" + b.NumericalID},
	}
	for _, e := range endpoints {
		r := call(t, e.method, e.path, "", nil)
		expectStatus(t, r, http.StatusUnauthorized)
		r = call(t, e.method, e.path, u.Token, nil)
		if expectStatus(t, r, http.StatusForbidden) {
			expectJSONError(t, r)
		}
	}

	// Promotion takes effect immediately on the existing token.
	setRole(t, u, "admin")
	r := call(t, "GET", "/auth/verify", u.Token, nil)
	if expectStatus(t, r, http.StatusOK) && !decode[verifyResponse](t, r).IsAdmin {
		t.Errorf("verify should report is_admin after promotion: %s", r)
	}
	expectStatus(t, call(t, "GET", "/admin/users?q="+posterPrefix, u.Token, nil), http.StatusOK)

	// ...and so does demotion.
	setRole(t, u, "user")
	expectStatus(t, call(t, "GET", "/admin/users?q="+posterPrefix, u.Token, nil), http.StatusForbidden)
}

func TestAdminSearchPosters(t *testing.T) {
	admin, target := newAdmin(t), newUser(t)
	b := newBike(t, target, "", false)
	createReview(t, target, b.NumericalID, map[string]any{"overall": 3})

	r := call(t, "GET", "/admin/users?q="+target.Username, admin.Token, nil)
	mustStatus(t, r, http.StatusOK)
	list := decode[[]posterSummary](t, r)
	if len(list) != 1 {
		t.Fatalf("search by username: %s", r)
	}
	p := list[0]
	if p.PosterID != target.ID || p.Email != target.Email || p.Role != "user" || p.ReviewCount != 1 || p.BikeCount != 1 {
		t.Errorf("poster summary: %s", r)
	}

	r = call(t, "GET", "/admin/users?q="+target.Email, admin.Token, nil)
	if expectStatus(t, r, http.StatusOK) && len(decode[[]posterSummary](t, r)) != 1 {
		t.Errorf("search by email: %s", r)
	}

	r = call(t, "GET", "/admin/users?q=zzzznomatch", admin.Token, nil)
	if expectStatus(t, r, http.StatusOK) && string(r.Body) != "[]\n" {
		t.Errorf("no match should be []: %s", r)
	}

	r = call(t, "GET", "/admin/users?q="+posterPrefix+"&limit=1", admin.Token, nil)
	if expectStatus(t, r, http.StatusOK) && len(decode[[]posterSummary](t, r)) != 1 {
		t.Errorf("limit=1: %s", r)
	}
}

func TestAdminPurgePoster(t *testing.T) {
	admin, target, bystander := newAdmin(t), newUser(t), newUser(t)
	targetBike := newBike(t, target, "", false)
	otherBike := newBike(t, bystander, "", false)

	onTargetBike := createReview(t, bystander, targetBike.NumericalID, map[string]any{"overall": 5})
	byTarget := createReview(t, target, otherBike.NumericalID, map[string]any{"overall": 1})
	createReview(t, bystander, otherBike.NumericalID, map[string]any{"overall": 5})
	expectAverage(t, otherBike.NumericalID, ptr(3.0))

	t.Run("guards", func(t *testing.T) {
		expectStatus(t, call(t, "DELETE", "/admin/users/abc", admin.Token, nil), http.StatusBadRequest)
		expectStatus(t, call(t, "DELETE", fmt.Sprintf("/admin/users/%d", admin.ID), admin.Token, nil), http.StatusBadRequest)
		expectStatus(t, call(t, "DELETE", "/admin/users/999999999", admin.Token, nil), http.StatusNotFound)
		otherAdmin := newAdmin(t)
		expectStatus(t, call(t, "DELETE", fmt.Sprintf("/admin/users/%d", otherAdmin.ID), admin.Token, nil), http.StatusBadRequest)
		expectStatus(t, call(t, "GET", "/auth/verify", otherAdmin.Token, nil), http.StatusOK)
	})

	mustStatus(t, call(t, "DELETE", fmt.Sprintf("/admin/users/%d", target.ID), admin.Token, nil), http.StatusNoContent)

	expectStatus(t, call(t, "GET", "/auth/verify", target.Token, nil), http.StatusUnauthorized)
	expectStatus(t, call(t, "GET", "/bikes/"+targetBike.NumericalID, "", nil), http.StatusNotFound)
	expectStatus(t, call(t, "GET", fmt.Sprintf("/reviews/%d", onTargetBike), "", nil), http.StatusNotFound)
	expectStatus(t, call(t, "GET", fmt.Sprintf("/reviews/%d", byTarget), "", nil), http.StatusNotFound)
	expectAverage(t, otherBike.NumericalID, ptr(5.0))
	expectStatus(t, call(t, "GET", "/auth/verify", bystander.Token, nil), http.StatusOK)

	r := call(t, "GET", "/admin/users?q="+target.Username, admin.Token, nil)
	if expectStatus(t, r, http.StatusOK) && string(r.Body) != "[]\n" {
		t.Errorf("purged poster still searchable: %s", r)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM moderation_actions WHERE admin_poster_id = $1 AND target_poster_id = $2 AND action = 'purge_poster'`,
		admin.ID, target.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected 1 moderation_actions audit row, found %d", n)
	}

	expectStatus(t, call(t, "DELETE", fmt.Sprintf("/admin/users/%d", target.ID), admin.Token, nil), http.StatusNotFound)
}

func TestAdminDeleteBike(t *testing.T) {
	admin, owner := newAdmin(t), newUser(t)
	b := newBike(t, owner, uniqueHash(), false)
	rid := createReview(t, owner, b.NumericalID, map[string]any{"overall": 2})

	expectStatus(t, call(t, "DELETE", "/admin/bikes/abc", admin.Token, nil), http.StatusBadRequest)
	expectStatus(t, call(t, "DELETE", "/admin/bikes/99999999", admin.Token, nil), http.StatusNotFound)

	mustStatus(t, call(t, "DELETE", "/admin/bikes/"+b.NumericalID, admin.Token, nil), http.StatusNoContent)

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM moderation_actions WHERE admin_poster_id = $1 AND target_bike_id = $2 AND action = 'delete_bike'`,
		admin.ID, b.NumericalID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected 1 moderation_actions audit row for the bike deletion, found %d", n)
	}
	expectStatus(t, call(t, "GET", "/bikes/"+b.NumericalID, "", nil), http.StatusNotFound)
	expectStatus(t, call(t, "GET", fmt.Sprintf("/reviews/%d", rid), "", nil), http.StatusNotFound)
	expectStatus(t, call(t, "GET", "/scan/"+*b.HashID, owner.Token, nil), http.StatusNotFound)
	expectStatus(t, call(t, "DELETE", "/admin/bikes/"+b.NumericalID, admin.Token, nil), http.StatusNotFound)

	// The id is free again.
	r := call(t, "POST", "/bikes", owner.Token, map[string]any{"numerical_id": b.NumericalID})
	expectStatus(t, r, http.StatusCreated)
}
