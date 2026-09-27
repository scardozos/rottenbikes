//go:build e2e

// Package e2e is a black-box end-to-end suite that runs against a live API
// (local, dev or prod). It talks to the API over HTTP and uses the database
// only for what the API deliberately never exposes or that a test must not
// trigger against a real environment:
//
//   - seeding test posters and their magic links for tests that aren't about
//     sign-up itself (the real flow needs a solved hCaptcha and sends an email);
//   - reading the emailed magic link when no Mailtrap sandbox inbox is
//     configured (see mail_test.go);
//   - promoting test posters to admin (done out-of-band by adminctl);
//   - a few post-condition checks (e.g. the moderation audit row);
//   - cleaning up everything the run created.
//
// Run it with `make e2e ENV=local|dev|prod` (see .scripts/run-e2e.sh).
package e2e

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lib/pq"
)

// Every poster the suite creates is named <posterPrefix><run><seq> with an
// @example.com email, so leftovers from aborted runs can be recognised.
const posterPrefix = "e2etest"

var (
	envName    = envOr("E2E_ENV", "local")
	apiURL     = strings.TrimRight(envOr("E2E_API_URL", "http://localhost:8080"), "/")
	dbDSN      = envOr("E2E_DB_DSN", "postgres://rottenbikes:rottenbikes@localhost:5432/rottenbikes?sslmode=disable")
	metricsURL = strings.TrimRight(os.Getenv("E2E_METRICS_URL"), "/") // optional; not exposed publicly
	corsOrigin = os.Getenv("E2E_CORS_ORIGIN")                         // optional; an origin the API allows

	db         *sql.DB
	httpClient = &http.Client{Timeout: 20 * time.Second}

	// captchaToken is a token the target's hCaptcha secret accepts: on dev,
	// which uses hCaptcha's test keys, the official test token.
	captchaToken = os.Getenv("E2E_CAPTCHA_TOKEN")
	// requireCaptcha makes the suite fail if captcha is not enforced (prod).
	requireCaptcha = os.Getenv("E2E_REQUIRE_CAPTCHA") == "1"

	// Captcha probe results (see probeCaptcha). When captcha is enforced and
	// captchaToken does not pass it, the tests that must call register /
	// request-magic-link are skipped.
	bogusCaptchaStatus, tokenCaptchaStatus int
	captchaEnforced, captchaPasses         bool

	runID   = strconv.FormatInt(time.Now().Unix()%1_000_000, 36) + randomHex(2)
	seq     atomic.Int64
	bikeSeq atomic.Int64

	created struct {
		sync.Mutex
		posters []int64
		bikes   []string
	}
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestMain(m *testing.M) {
	var err error
	db, err = sql.Open("postgres", dbDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: open db: %v\n", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: cannot reach the %s database: %v\n", envName, err)
		os.Exit(1)
	}
	resp, err := httpClient.Get(apiURL + "/readyz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: cannot reach the %s API at %s: %v\n", envName, apiURL, err)
		os.Exit(1)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "e2e: %s/readyz returned %d\n", apiURL, resp.StatusCode)
		os.Exit(1)
	}

	sweepStale()
	bogusCaptchaStatus = probeCaptcha("e2e-bogus-captcha")
	captchaEnforced = bogusCaptchaStatus != http.StatusBadRequest
	captchaPasses = !captchaEnforced
	if captchaToken != "" {
		tokenCaptchaStatus = probeCaptcha(captchaToken)
		captchaPasses = tokenCaptchaStatus == http.StatusBadRequest
	}
	bikeSeq.Store(10000 + time.Now().UnixNano()%80000)

	fmt.Printf("e2e: env=%s api=%s run=%s captcha_enforced=%v captcha_passes=%v mail=%s\n",
		envName, apiURL, runID, captchaEnforced, captchaPasses, mailSource())

	code := m.Run()
	cleanup()
	db.Close()
	os.Exit(code)
}

// probeCaptcha sends a register request with the given captcha token and an
// invalid email, and returns the status. The captcha is checked first, so
// 400 means the captcha passed (and nothing was created), 403 that it was
// rejected and 503 that it could not be verified.
func probeCaptcha(token string) int {
	raw, _ := json.Marshal(map[string]string{
		"username": posterPrefix + "probe", "email": "not-an-email", "captcha_token": token,
	})
	resp, err := httpClient.Post(apiURL+"/auth/register", "application/json", bytes.NewReader(raw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: captcha probe: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// cleanup deletes everything this run created: bikes (cascading to their
// reviews, ratings, aggregates and scans), audit rows and posters (cascading
// to their sessions and magic links).
func cleanup() {
	created.Lock()
	defer created.Unlock()
	if err := purge(created.posters, created.bikes); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: CLEANUP FAILED, run %s may have left data behind: %v\n", runID, err)
	}
}

// sweepStale removes leftovers of aborted runs older than an hour (bikes
// orphaned by account deletion cannot be traced back and are not swept).
func sweepStale() {
	rows, err := db.Query(`SELECT poster_id FROM posters
		WHERE username LIKE $1 AND email LIKE '%@example.com' AND created_ts < NOW() - INTERVAL '1 hour'`,
		posterPrefix+"%")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: sweep stale: %v\n", err)
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	if len(ids) == 0 {
		return
	}
	if err := purge(ids, nil); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: sweep stale: %v\n", err)
		return
	}
	fmt.Printf("e2e: swept %d stale test posters from earlier runs\n", len(ids))
}

func purge(posters []int64, bikes []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmts := []struct {
		query string
		arg   any
	}{
		{`DELETE FROM bikes WHERE numerical_id = ANY($1)`, pq.Array(bikes)},
		{`DELETE FROM bikes WHERE creator_id = ANY($1)`, pq.Array(posters)},
		{`DELETE FROM reviews WHERE poster_id = ANY($1)`, pq.Array(posters)},
		{`DELETE FROM moderation_actions WHERE admin_poster_id = ANY($1) OR target_poster_id = ANY($1)`, pq.Array(posters)},
		{`DELETE FROM posters WHERE poster_id = ANY($1)`, pq.Array(posters)},
	}
	for i, s := range stmts {
		res, err := tx.Exec(s.query, s.arg)
		if err != nil {
			return fmt.Errorf("%s: %w", s.query, err)
		}
		// Tests only review bikes they created, which are already gone by now.
		if n, _ := res.RowsAffected(); i == 2 && n > 0 {
			fmt.Fprintf(os.Stderr, "e2e: warning: deleted %d test reviews on non-test bikes; their aggregates may be stale\n", n)
		}
	}
	return tx.Commit()
}

func trackPoster(id int64) {
	created.Lock()
	created.posters = append(created.posters, id)
	created.Unlock()
}

func trackBike(id string) {
	created.Lock()
	created.bikes = append(created.bikes, id)
	created.Unlock()
}

// requireCaptchaPass skips tests that must get past the captcha when the
// suite has no way to do so on this environment.
func requireCaptchaPass(t *testing.T) {
	t.Helper()
	if !captchaPasses {
		t.Skipf("hCaptcha is enforced on %s and no accepted E2E_CAPTCHA_TOKEN is configured", envName)
	}
}

// passingCaptcha is the captcha_token to send when the request should get
// past the captcha.
func passingCaptcha() string {
	if captchaToken != "" {
		return captchaToken
	}
	return "e2e"
}

// ---------------------------------------------------------------------------
// HTTP
// ---------------------------------------------------------------------------

type response struct {
	Method string
	Path   string
	Status int
	Header http.Header
	Body   []byte
}

func (r response) String() string {
	return fmt.Sprintf("%s %s -> %d %s", r.Method, r.Path, r.Status, strings.TrimSpace(string(r.Body)))
}

type reqOpt func(*http.Request)

func withHeader(k, v string) reqOpt {
	return func(r *http.Request) { r.Header.Set(k, v) }
}

// call sends a request. body may be nil, a string (sent raw) or any value
// (JSON-encoded). token, if non-empty, is sent as a Bearer token. Bikes and
// posters created through the API are tracked for cleanup.
func call(t *testing.T, method, path, token string, body any, opts ...reqOpt) response {
	t.Helper()
	var raw []byte
	switch b := body.(type) {
	case nil:
	case string:
		raw = []byte(b)
	default:
		var err error
		if raw, err = json.Marshal(b); err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	var rdr io.Reader
	if raw != nil {
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, apiURL+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if rdr != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for _, o := range opts {
		o(req)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	r := response{Method: method, Path: path, Status: resp.StatusCode, Header: resp.Header, Body: out}
	track(t, r, raw)
	return r
}

func track(t *testing.T, r response, reqBody []byte) {
	t.Helper()
	switch {
	case r.Method == "POST" && r.Path == "/bikes" && r.Status == http.StatusCreated:
		var b bike
		if json.Unmarshal(r.Body, &b) == nil && b.NumericalID != "" {
			trackBike(b.NumericalID)
		}
	case r.Method == "POST" && r.Path == "/auth/register" && r.Status == http.StatusOK:
		var req struct {
			Email string `json:"email"`
		}
		_ = json.Unmarshal(reqBody, &req)
		var id int64
		if err := db.QueryRow(`SELECT poster_id FROM posters WHERE email = $1`, req.Email).Scan(&id); err != nil {
			t.Errorf("track registered poster %q: %v", req.Email, err)
			return
		}
		trackPoster(id)
	}
}

// expectStatus reports (non-fatally) a status mismatch and returns whether it matched.
func expectStatus(t *testing.T, r response, want int) bool {
	t.Helper()
	if r.Status != want {
		t.Errorf("expected %d, got: %s", want, r)
		return false
	}
	return true
}

// mustStatus is like expectStatus but stops the test on mismatch; use it when
// later steps depend on the call having succeeded.
func mustStatus(t *testing.T, r response, want int) {
	t.Helper()
	if r.Status != want {
		t.Fatalf("expected %d, got: %s", want, r)
	}
}

func decode[T any](t *testing.T, r response) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatalf("decode %s %s body %q: %v", r.Method, r.Path, r.Body, err)
	}
	return v
}

// expectJSONError checks the uniform {"error": "..."} error envelope.
func expectJSONError(t *testing.T, r response) {
	t.Helper()
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("error response should be JSON, got Content-Type %q: %s", ct, r)
		return
	}
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(r.Body, &e); err != nil || e.Error == "" {
		t.Errorf(`error response should be {"error": "..."}: %s`, r)
	}
}

// ---------------------------------------------------------------------------
// API shapes
// ---------------------------------------------------------------------------

type bike struct {
	NumericalID   string    `json:"numerical_id"`
	HashID        *string   `json:"hash_id"`
	IsElectric    bool      `json:"is_electric"`
	WasScanned    bool      `json:"was_scanned"`
	AverageRating *float64  `json:"average_rating"`
	CreatedTS     time.Time `json:"created_ts"`
	UpdatedTS     time.Time `json:"updated_ts"`
}

type review struct {
	ReviewID        int64            `json:"review_id"`
	PosterID        int64            `json:"poster_id"`
	PosterUsername  string           `json:"poster_username"`
	BikeNumericalID string           `json:"bike_numerical_id"`
	Comment         *string          `json:"comment"`
	CreatedAt       time.Time        `json:"created_at"`
	Ratings         map[string]int16 `json:"ratings"`
	BikeImg         *string          `json:"bike_img"`
	WasScanned      bool             `json:"was_scanned"`
}

type ratingAggregate struct {
	BikeNumericalID string  `json:"bike_numerical_id"`
	Subcategory     string  `json:"subcategory"`
	AverageRating   float64 `json:"average_rating"`
	Window          string  `json:"window"`
}

type bikeDetails struct {
	bike
	Ratings      []ratingAggregate `json:"ratings"`
	Reviews      []review          `json:"reviews"`
	TotalReviews int               `json:"total_reviews"`
}

type posterSummary struct {
	PosterID    int64  `json:"poster_id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	ReviewCount int64  `json:"review_count"`
	BikeCount   int64  `json:"bike_count"`
}

type verifyResponse struct {
	PosterID int64  `json:"poster_id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	Status   string `json:"status"`
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

type user struct {
	ID       int64
	Username string
	Email    string
	Token    string
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func uniqueUsername() string {
	return fmt.Sprintf("%s%s%d", posterPrefix, runID, seq.Add(1))
}

// seedPoster inserts an unverified poster, as POST /auth/register would.
func seedPoster(t *testing.T) user {
	t.Helper()
	name := uniqueUsername()
	u := user{Username: name, Email: name + "@example.com"}
	if err := db.QueryRow(`INSERT INTO posters (email, username) VALUES ($1, $2) RETURNING poster_id`,
		u.Email, u.Username).Scan(&u.ID); err != nil {
		t.Fatalf("seed poster: %v", err)
	}
	trackPoster(u.ID)
	return u
}

// seedMagicLink issues a magic link for the poster, as register /
// request-magic-link would, and returns the raw emailed token and poll token.
func seedMagicLink(t *testing.T, posterID int64) (magic, poll string) {
	t.Helper()
	magic, poll = randomHex(32), randomHex(32)
	if _, err := db.Exec(`INSERT INTO magic_links (poster_id, token, poll_token, expires_ts)
		VALUES ($1, $2, $3, NOW() + INTERVAL '30 minutes')`, posterID, sha256Hex(magic), sha256Hex(poll)); err != nil {
		t.Fatalf("seed magic link: %v", err)
	}
	return magic, poll
}

// interceptMagicLink stands in for reading the email sent by a real
// register / request-magic-link call: the API only stores the SHA-256 of the
// emailed token, so we swap it for the hash of a token we know.
func interceptMagicLink(t *testing.T, pollToken string) string {
	t.Helper()
	magic := randomHex(32)
	res, err := db.Exec(`UPDATE magic_links SET token = $1 WHERE poll_token = $2`, sha256Hex(magic), sha256Hex(pollToken))
	if err != nil {
		t.Fatalf("intercept magic link: %v", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		t.Fatalf("intercept magic link: expected 1 magic_links row for poll token, got %d", n)
	}
	return magic
}

// registerViaAPI registers a fresh poster through POST /auth/register and
// returns it with the raw poll token.
func registerViaAPI(t *testing.T) (u user, pollToken string) {
	t.Helper()
	requireCaptchaPass(t)
	name := uniqueUsername()
	u = user{Username: name, Email: name + "@example.com"}
	r := call(t, "POST", "/auth/register", "", map[string]string{
		"username": u.Username, "email": u.Email, "captcha_token": passingCaptcha(),
	})
	mustStatus(t, r, http.StatusOK)
	body := decode[map[string]string](t, r)
	if body["magic_token"] == "" {
		t.Fatalf("register returned no poll token: %s", r)
	}
	return u, body["magic_token"]
}

// confirm confirms a magic token via the API and returns the api token.
func confirm(t *testing.T, magic string) string {
	t.Helper()
	r := call(t, "GET", "/auth/confirm/"+magic, "", nil)
	mustStatus(t, r, http.StatusOK)
	body := decode[map[string]any](t, r)
	tok, _ := body["api_token"].(string)
	if tok == "" {
		t.Fatalf("confirm returned no api_token: %s", r)
	}
	return tok
}

// newUser seeds a poster and logs it in through the API.
func newUser(t *testing.T) *user {
	t.Helper()
	u := seedPoster(t)
	magic, _ := seedMagicLink(t, u.ID)
	u.Token = confirm(t, magic)
	return &u
}

func setRole(t *testing.T, u *user, role string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE posters SET role = $1::poster_role WHERE poster_id = $2`, role, u.ID); err != nil {
		t.Fatalf("set role: %v", err)
	}
}

func newAdmin(t *testing.T) *user {
	t.Helper()
	u := newUser(t)
	setRole(t, u, "admin")
	return u
}

// nextBikeID returns a candidate 5-digit id. It may already be taken in a
// shared environment, so callers creating bikes retry on 409.
func nextBikeID() string {
	return strconv.FormatInt(10000+bikeSeq.Add(1)%90000, 10)
}

// newBike creates a bike owned by u, retrying on id collisions.
func newBike(t *testing.T, u *user, hash string, electric bool) bike {
	t.Helper()
	for i := 0; i < 20; i++ {
		body := map[string]any{"numerical_id": nextBikeID(), "is_electric": electric}
		if hash != "" {
			body["hash_id"] = hash
		}
		r := call(t, "POST", "/bikes", u.Token, body)
		if r.Status == http.StatusConflict {
			continue
		}
		mustStatus(t, r, http.StatusCreated)
		return decode[bike](t, r)
	}
	t.Fatalf("could not allocate a free bike id")
	return bike{}
}

// freeBikeID returns an id that no bike currently uses.
func freeBikeID(t *testing.T) string {
	t.Helper()
	for i := 0; i < 20; i++ {
		id := nextBikeID()
		var exists bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM bikes WHERE numerical_id = $1)`, id).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			return id
		}
	}
	t.Fatalf("could not find a free bike id")
	return ""
}

func uniqueHash() string {
	return "e2e" + randomHex(8)
}

func ptr[T any](v T) *T { return &v }

// createReview posts a review and returns its id.
func createReview(t *testing.T, u *user, bikeID string, body map[string]any) int64 {
	t.Helper()
	r := call(t, "POST", "/bikes/"+bikeID+"/reviews", u.Token, body)
	mustStatus(t, r, http.StatusCreated)
	id := decode[map[string]int64](t, r)["review_id"]
	if id == 0 {
		t.Fatalf("create review returned no review_id: %s", r)
	}
	return id
}

func getBike(t *testing.T, id string) bike {
	t.Helper()
	r := call(t, "GET", "/bikes/"+id, "", nil)
	mustStatus(t, r, http.StatusOK)
	return decode[bike](t, r)
}

func expectAverage(t *testing.T, id string, want *float64) {
	t.Helper()
	got := getBike(t, id).AverageRating
	switch {
	case want == nil && got != nil:
		t.Errorf("bike %s: expected no average_rating, got %v", id, *got)
	case want != nil && got == nil:
		t.Errorf("bike %s: expected average_rating %v, got null", id, *want)
	case want != nil && got != nil && *got != *want:
		t.Errorf("bike %s: expected average_rating %v, got %v", id, *want, *got)
	}
}
