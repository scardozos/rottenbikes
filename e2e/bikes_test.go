//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestCreateBike(t *testing.T) {
	u := newUser(t)

	t.Run("requires auth", func(t *testing.T) {
		r := call(t, "POST", "/bikes", "", map[string]any{"numerical_id": nextBikeID()})
		if expectStatus(t, r, http.StatusUnauthorized) {
			expectJSONError(t, r)
		}
	})

	t.Run("validation", func(t *testing.T) {
		for _, id := range []string{"", "123", "1234567", "12a456", "-12345", " 12345"} {
			r := call(t, "POST", "/bikes", u.Token, map[string]any{"numerical_id": id})
			if expectStatus(t, r, http.StatusBadRequest) {
				expectJSONError(t, r)
			}
		}
		r := call(t, "POST", "/bikes", u.Token, map[string]any{"numerical_id": nextBikeID(), "hash_id": "not-alnum!"})
		expectStatus(t, r, http.StatusBadRequest)
		r = call(t, "POST", "/bikes", u.Token, "{bad")
		expectStatus(t, r, http.StatusBadRequest)
	})

	// Test accounts can only use the reserved 6-digit range and regular
	// accounts only real 4-5 digit numbers, so on shared environments the suite
	// can never take (or clean up) a real bike's number. Only rejected
	// requests here, so this is safe in every mode.
	t.Run("reserved number ranges", func(t *testing.T) {
		tester := newUserWith(t, true)
		for _, id := range []string{"1234", "12345", "01234"} {
			r := call(t, "POST", "/bikes", tester.Token, map[string]any{"numerical_id": id})
			if expectStatus(t, r, http.StatusBadRequest) {
				expectJSONError(t, r)
			}
		}
		regular := newUserWith(t, false)
		r := call(t, "POST", "/bikes", regular.Token, map[string]any{"numerical_id": freeBikeIDFor(t, true)})
		expectStatus(t, r, http.StatusBadRequest)
	})

	t.Run("create and read back", func(t *testing.T) {
		id, hash := freeBikeID(t), uniqueHash()
		r := call(t, "POST", "/bikes", u.Token, map[string]any{
			"numerical_id": id, "hash_id": hash, "is_electric": true, "was_scanned": true,
		})
		mustStatus(t, r, http.StatusCreated)
		b := decode[bike](t, r)
		if b.NumericalID != id || b.HashID == nil || *b.HashID != hash || !b.IsElectric || !b.WasScanned || b.AverageRating != nil {
			t.Errorf("created bike mismatch: %s", r)
		}
		got := getBike(t, id)
		if got.NumericalID != id || got.HashID == nil || *got.HashID != hash || !got.IsElectric || !got.WasScanned {
			t.Errorf("GET bike mismatch: %+v", got)
		}

		// Leading zeros are preserved (numerical_id is a string).
		zid := "0" + freeBikeID(t)[1:]
		r = call(t, "POST", "/bikes", u.Token, map[string]any{"numerical_id": zid})
		if r.Status != http.StatusConflict && expectStatus(t, r, http.StatusCreated) {
			if getBike(t, zid).NumericalID != zid {
				t.Errorf("leading zero not preserved for %s", zid)
			}
		}

		// Duplicates.
		r = call(t, "POST", "/bikes", u.Token, map[string]any{"numerical_id": id})
		if expectStatus(t, r, http.StatusConflict) {
			expectJSONError(t, r)
		}
		r = call(t, "POST", "/bikes", u.Token, map[string]any{"numerical_id": freeBikeID(t), "hash_id": hash})
		expectStatus(t, r, http.StatusConflict)
	})

	t.Run("empty hash_id is treated as absent", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			r := call(t, "POST", "/bikes", u.Token, map[string]any{"numerical_id": freeBikeID(t), "hash_id": ""})
			if expectStatus(t, r, http.StatusCreated) && decode[bike](t, r).HashID != nil {
				t.Errorf("expected hash_id null: %s", r)
			}
		}
	})
}

func TestGetAndListBikes(t *testing.T) {
	u := newUser(t)
	b1 := newBike(t, u, uniqueHash(), false)
	b2 := newBike(t, u, "", true)

	t.Run("get", func(t *testing.T) {
		expectStatus(t, call(t, "GET", "/bikes/"+b1.NumericalID, "", nil), http.StatusOK)
		r := call(t, "GET", "/bikes/99999999", "", nil)
		if expectStatus(t, r, http.StatusNotFound) {
			expectJSONError(t, r)
		}
		expectStatus(t, call(t, "GET", "/bikes/abc", "", nil), http.StatusBadRequest)
	})

	// Listed as the test account that created them: test bikes are hidden from
	// everyone else (see TestTestBikesAreHidden).
	t.Run("list and search", func(t *testing.T) {
		r := call(t, "GET", "/bikes", u.Token, nil)
		mustStatus(t, r, http.StatusOK)
		all := decode[[]bike](t, r)
		found := 0
		for _, b := range all {
			if b.NumericalID == b1.NumericalID || b.NumericalID == b2.NumericalID {
				found++
			}
		}
		if found != 2 {
			t.Errorf("GET /bikes: expected both new bikes in the list, found %d", found)
		}

		// Search is a substring match on numerical_id and hash_id, so in a
		// shared environment other bikes may match the id too.
		r = call(t, "GET", "/bikes?q="+b1.NumericalID, u.Token, nil)
		mustStatus(t, r, http.StatusOK)
		hits := decode[[]bike](t, r)
		found = 0
		for _, b := range hits {
			if b.NumericalID == b1.NumericalID {
				found++
			}
		}
		if found != 1 {
			t.Errorf("search by numerical_id: %s", r)
		}

		r = call(t, "GET", "/bikes?q="+*b1.HashID, u.Token, nil)
		mustStatus(t, r, http.StatusOK)
		hits = decode[[]bike](t, r)
		if len(hits) != 1 || hits[0].NumericalID != b1.NumericalID {
			t.Errorf("search by hash_id: %s", r)
		}

		r = call(t, "GET", "/bikes?q=zzzznomatch", u.Token, nil)
		if expectStatus(t, r, http.StatusOK) && string(r.Body) != "[]\n" {
			t.Errorf("empty search should return []: %s", r)
		}

		r = call(t, "GET", "/bikes?limit=1", u.Token, nil)
		if expectStatus(t, r, http.StatusOK) && len(decode[[]bike](t, r)) != 1 {
			t.Errorf("limit=1: %s", r)
		}

		for _, sort := range []string{"rating", "most_reviewed", "recent", "bogus"} {
			expectStatus(t, call(t, "GET", "/bikes?sort="+sort, u.Token, nil), http.StatusOK)
		}
	})
}

func TestScanBike(t *testing.T) {
	u := newUser(t)
	hash := uniqueHash()
	b := newBike(t, u, hash, false)

	expectStatus(t, call(t, "GET", "/scan/"+hash, "", nil), http.StatusUnauthorized)
	r := call(t, "GET", "/scan/"+hash, u.Token, nil)
	if expectStatus(t, r, http.StatusOK) && decode[bike](t, r).NumericalID != b.NumericalID {
		t.Errorf("scan returned wrong bike: %s", r)
	}
	expectStatus(t, call(t, "GET", "/scan/"+uniqueHash(), u.Token, nil), http.StatusNotFound)
	expectStatus(t, call(t, "GET", "/scan/bad-hash", u.Token, nil), http.StatusBadRequest)
}

func TestUpdateBike(t *testing.T) {
	owner, other := newUser(t), newUser(t)
	b := newBike(t, owner, uniqueHash(), false)
	path := "/bikes/" + b.NumericalID

	t.Run("owner can update", func(t *testing.T) {
		newHash := uniqueHash()
		r := call(t, "PUT", path, owner.Token, map[string]any{"hash_id": newHash, "is_electric": true})
		mustStatus(t, r, http.StatusNoContent)
		got := getBike(t, b.NumericalID)
		if got.HashID == nil || *got.HashID != newHash || !got.IsElectric {
			t.Errorf("update not applied: %+v", got)
		}
		if !got.UpdatedTS.After(b.UpdatedTS) {
			t.Errorf("updated_ts not bumped: before %v after %v", b.UpdatedTS, got.UpdatedTS)
		}

		// Partial update keeps other fields.
		mustStatus(t, call(t, "PUT", path, owner.Token, map[string]any{"is_electric": false}), http.StatusNoContent)
		got = getBike(t, b.NumericalID)
		if got.HashID == nil || *got.HashID != newHash || got.IsElectric {
			t.Errorf("partial update clobbered fields: %+v", got)
		}
	})

	t.Run("non-owner and missing bike are 404", func(t *testing.T) {
		expectStatus(t, call(t, "PUT", path, other.Token, map[string]any{"is_electric": true}), http.StatusNotFound)
		expectStatus(t, call(t, "PUT", "/bikes/99999999", owner.Token, map[string]any{"is_electric": true}), http.StatusNotFound)
		expectStatus(t, call(t, "PUT", path, "", map[string]any{"is_electric": true}), http.StatusUnauthorized)
	})

	t.Run("validation", func(t *testing.T) {
		expectStatus(t, call(t, "PUT", path, owner.Token, map[string]any{"numerical_id": 1234}), http.StatusBadRequest)
		expectStatus(t, call(t, "PUT", path, owner.Token, map[string]any{"hash_id": "bad-hash"}), http.StatusBadRequest)
		expectStatus(t, call(t, "PUT", path, owner.Token, "{bad"), http.StatusBadRequest)
		expectStatus(t, call(t, "PUT", "/bikes/abc", owner.Token, map[string]any{}), http.StatusBadRequest)
	})

	t.Run("hash_id already used by another bike is a conflict", func(t *testing.T) {
		taken := uniqueHash()
		newBike(t, owner, taken, false)
		r := call(t, "PUT", path, owner.Token, map[string]any{"hash_id": taken})
		expectStatus(t, r, http.StatusConflict)
	})

	t.Run("empty hash_id clears it, like on create", func(t *testing.T) {
		b1 := newBike(t, owner, uniqueHash(), false)
		b2 := newBike(t, owner, uniqueHash(), false)
		for _, x := range []bike{b1, b2} {
			r := call(t, "PUT", "/bikes/"+x.NumericalID, owner.Token, map[string]any{"hash_id": ""})
			if !expectStatus(t, r, http.StatusNoContent) {
				continue
			}
			if got := getBike(t, x.NumericalID); got.HashID != nil {
				t.Errorf("bike %s: expected hash_id null after clearing, got %q", x.NumericalID, *got.HashID)
			}
		}
	})
}

// Bikes created by test accounts (the suite's) must not show up in anyone
// else's listings or search, so running the suite against a shared
// environment doesn't show real users test data. They stay reachable by id.
func TestTestBikesAreHidden(t *testing.T) {
	// Explicit accounts, so this also runs (and checks the hiding) in the
	// local mode where the suite otherwise uses regular accounts.
	tester := newUserWith(t, true)
	realUser := newUserWith(t, false)
	hash := uniqueHash()
	b := newBike(t, tester, hash, false)

	var isTest bool
	if err := db.QueryRow(`SELECT is_test FROM bikes WHERE numerical_id = $1`, b.NumericalID).Scan(&isTest); err != nil {
		t.Fatal(err)
	}
	if !isTest {
		t.Fatalf("a bike created by a test account must be flagged as a test bike")
	}

	listed := func(token string) bool {
		t.Helper()
		found := false
		for _, q := range []string{"/bikes?q=" + hash, "/bikes?q=" + b.NumericalID, "/bikes?sort=recent&limit=100"} {
			r := call(t, "GET", q, token, nil)
			mustStatus(t, r, http.StatusOK)
			for _, x := range decode[[]bike](t, r) {
				if x.NumericalID == b.NumericalID {
					found = true
				}
			}
		}
		return found
	}

	check := func(t *testing.T) {
		if listed("") {
			t.Error("test bike listed for anonymous visitors")
		}
		if listed(realUser.Token) {
			t.Error("test bike listed for a regular account")
		}
		if listed("invalid-token") {
			t.Error("test bike listed for an invalid token")
		}
		expectStatus(t, call(t, "GET", "/bikes/"+b.NumericalID, "", nil), http.StatusOK)
	}

	t.Run("hidden from listings", func(t *testing.T) {
		check(t)
		if !listed(tester.Token) {
			t.Error("test bike should be listed for test accounts")
		}
	})

	// The flag belongs to the bike, so it stays hidden once its creator is
	// gone and the bike is orphaned.
	t.Run("still hidden after the creator deletes their account", func(t *testing.T) {
		mustStatus(t, call(t, "DELETE", "/auth/user", tester.Token, nil), http.StatusNoContent)
		check(t)
	})
}
