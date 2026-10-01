package domain

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func TestValidationErrorsMatchErrValidation(t *testing.T) {
	err := validationErrorf("bad %s", "thing")
	if !errors.Is(err, ErrValidation) {
		t.Fatal("ValidationError should match ErrValidation")
	}
	if err.Error() != "bad thing" {
		t.Errorf("message %q", err.Error())
	}
}

func TestValidateEmail(t *testing.T) {
	for _, ok := range []string{"a@example.com", "A.B+c_d%e-f@sub.example.co", "UPPER@EXAMPLE.COM"} {
		if err := validateEmail(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	// Includes forms net/mail accepts but the posters.email_valid CHECK rejects.
	for _, bad := range []string{"", "not-an-email", "a@b", "a@b.c", "Alice <a@example.com>", "a b@example.com", "a@exa mple.com"} {
		if err := validateEmail(bad); !errors.Is(err, ErrValidation) {
			t.Errorf("%q should be invalid, got %v", bad, err)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	for _, ok := range []string{"alice", "a.b", "A1"} {
		if err := validateUsername(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "bad user", "bad!", "ñandu", "a_b"} {
		if err := validateUsername(bad); !errors.Is(err, ErrValidation) {
			t.Errorf("%q should be invalid, got %v", bad, err)
		}
	}
}

func TestValidateComment(t *testing.T) {
	ok := strings.Repeat("é", MaxReviewCommentLength) // 500 chars, 1000 bytes
	if err := validateComment(&ok); err != nil {
		t.Errorf("500 multibyte characters should be valid: %v", err)
	}
	if err := validateComment(nil); err != nil {
		t.Errorf("nil comment should be valid: %v", err)
	}
	tooLong := strings.Repeat("x", MaxReviewCommentLength+1)
	if err := validateComment(&tooLong); !errors.Is(err, ErrValidation) {
		t.Errorf("501 characters should be invalid, got %v", err)
	}
}

func TestValidateReviewScores(t *testing.T) {
	five, zero, six := int16(5), int16(0), int16(6)
	if err := validateReviewScores(&five, nil, nil, nil, nil, &five); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := validateReviewScores(&five, &zero, nil, nil, nil, nil); !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "breaks") {
		t.Errorf("expected breaks validation error, got %v", err)
	}
	if err := validateReviewScores(nil, nil, nil, nil, nil, &six); !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "pedals") {
		t.Errorf("expected pedals validation error, got %v", err)
	}
}

func TestValidateNumericalID(t *testing.T) {
	cases := []struct {
		id          string
		testAccount bool
		ok          bool
	}{
		{"1234", false, true},
		{"12345", false, true},
		{"01234", false, true},
		{"123", false, false},
		{"123456", false, false}, // reserved for test accounts
		{"12a4", false, false},
		{"", false, false},
		{"123456", true, true},
		{"012345", true, true},
		{"1234", true, false}, // real bike numbers are off limits for tests
		{"12345", true, false},
		{"1234567", true, false},
		{"12a456", true, false},
	}
	for _, c := range cases {
		err := ValidateNumericalID(c.id, c.testAccount)
		if c.ok && err != nil {
			t.Errorf("%q (test account: %v) should be valid: %v", c.id, c.testAccount, err)
		}
		if !c.ok && !errors.Is(err, ErrValidation) {
			t.Errorf("%q (test account: %v) should be invalid, got %v", c.id, c.testAccount, err)
		}
	}
}

// Invalid input is rejected before touching the database.
func TestServiceValidationBeforeDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := NewService(NewStore(db))
	ctx := context.Background()

	if _, err := svc.Register(ctx, "alice", "Alice <a@example.com>"); !errors.Is(err, ErrValidation) {
		t.Errorf("Register invalid email: %v", err)
	}
	if _, err := svc.Register(ctx, "bad user", "a@example.com"); !errors.Is(err, ErrValidation) {
		t.Errorf("Register invalid username: %v", err)
	}
	six := int16(6)
	long := strings.Repeat("x", 501)
	if _, err := svc.CreateReviewWithRatings(ctx, CreateReviewInput{PosterID: 1, BikeID: "1234", Overall: &six}); !errors.Is(err, ErrValidation) {
		t.Errorf("CreateReview invalid score: %v", err)
	}
	if _, err := svc.CreateReviewWithRatings(ctx, CreateReviewInput{PosterID: 1, BikeID: "1234", Comment: &long}); !errors.Is(err, ErrValidation) {
		t.Errorf("CreateReview long comment: %v", err)
	}
	if err := svc.UpdateReviewWithRatings(ctx, UpdateReviewInput{ReviewID: 1, PosterID: 1, Comment: &long}); !errors.Is(err, ErrValidation) {
		t.Errorf("UpdateReview long comment: %v", err)
	}
	if _, err := svc.CreateBike(ctx, "12", nil, false, false, 1, false); !errors.Is(err, ErrValidation) {
		t.Errorf("CreateBike invalid id: %v", err)
	}
	bad := "not-alnum!"
	if err := svc.UpdateBike(ctx, "1234", &bad, nil, 1); !errors.Is(err, ErrValidation) {
		t.Errorf("UpdateBike invalid hash: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCreateReviewUnknownBike(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	score := int16(3)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT poster_id FROM posters").WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"poster_id"}).AddRow(1))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM reviews").WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT created_ts FROM reviews").WithArgs(int64(1), "9999").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("INSERT INTO reviews").
		WillReturnError(&pq.Error{Code: "23503", Constraint: "fk_reviews_bike"})
	mock.ExpectRollback()

	_, err = NewService(NewStore(db)).CreateReviewWithRatings(context.Background(), CreateReviewInput{PosterID: 1, BikeID: "9999", Overall: &score})
	if !errors.Is(err, ErrBikeNotFound) {
		t.Errorf("expected ErrBikeNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestAdminDeleteBike(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes and audits in one transaction", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		defer db.Close()
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM bikes").WithArgs("1234").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO moderation_actions").WithArgs(int64(7), "delete_bike", "1234").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		if err := NewService(NewStore(db)).AdminDeleteBike(ctx, 7, "1234"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("missing bike is not audited", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		defer db.Close()
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM bikes").WithArgs("1234").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		if err := NewService(NewStore(db)).AdminDeleteBike(ctx, 7, "1234"); err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("out-of-band deletion without audit", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		defer db.Close()
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM bikes").WithArgs("1234").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		if err := NewStore(db).AdminDeleteBike(ctx, "1234", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

func TestWindowedAggregatesEmptyIsNotNil(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery("SELECT").WithArgs("1234").
		WillReturnRows(sqlmock.NewRows([]string{"subcategory", "avg_1w", "avg_2w", "avg_overall"}))

	aggs, err := NewStore(db).ListWindowedRatingAggregatesByBike(context.Background(), "1234")
	if err != nil {
		t.Fatal(err)
	}
	if aggs == nil {
		t.Error("expected an empty, non-nil slice so it serializes as []")
	}
}
