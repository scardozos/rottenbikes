package domain

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"
)

// ErrValidation matches (via errors.Is) every *ValidationError, so callers can
// map invalid client input to a 400 instead of an internal error.
var ErrValidation = errors.New("validation error")

// ValidationError is invalid client input. Its message is safe to return to
// the client.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

func (e *ValidationError) Is(target error) bool { return target == ErrValidation }

func validationErrorf(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

// ErrBikeNotFound is returned when an operation references a bike that does
// not exist.
var ErrBikeNotFound = errors.New("bike not found")

// MaxReviewCommentLength matches reviews.comment VARCHAR(500) (characters).
const MaxReviewCommentLength = 500

var (
	validUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9.]+$`)
	// Same pattern as the posters.email_valid CHECK constraint, so anything
	// accepted here is also accepted by the database.
	validEmailRegex = regexp.MustCompile(`(?i)^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)
)

func validateEmail(email string) error {
	if !validEmailRegex.MatchString(email) {
		return validationErrorf("invalid email format")
	}
	return nil
}

func validateUsername(username string) error {
	if !validUsernameRegex.MatchString(username) {
		return validationErrorf("username can only contain letters, numbers and dots")
	}
	return nil
}

func validateComment(comment *string) error {
	if comment != nil && utf8.RuneCountInString(*comment) > MaxReviewCommentLength {
		return validationErrorf("comment must be at most %d characters", MaxReviewCommentLength)
	}
	return nil
}

func validateScore(sub RatingSubcategory, val *int16) error {
	if val == nil {
		return nil
	}
	if *val < 1 || *val > 5 {
		return validationErrorf("invalid score %d for %s: must be between 1 and 5", *val, sub)
	}
	return nil
}

func validateReviewScores(overall, breaks, seat, sturdiness, power, pedals *int16) error {
	for _, s := range []struct {
		sub RatingSubcategory
		val *int16
	}{
		{RatingSubcategoryOverall, overall},
		{RatingSubcategoryBreaks, breaks},
		{RatingSubcategorySeat, seat},
		{RatingSubcategorySturdiness, sturdiness},
		{RatingSubcategoryPower, power},
		{RatingSubcategoryPedals, pedals},
	} {
		if err := validateScore(s.sub, s.val); err != nil {
			return err
		}
	}
	return nil
}

func validateNumericalID(id string) error {
	if len(id) < 4 || len(id) > 5 {
		return validationErrorf("numerical_id must be 4-5 digits")
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return validationErrorf("numerical_id must be 4-5 digits")
		}
	}
	return nil
}

func validateHashID(hashID *string) error {
	if hashID != nil && *hashID != "" && !isDomainAlphanumeric(*hashID) {
		return validationErrorf("hash_id must be alphanumeric")
	}
	return nil
}

func isDomainAlphanumeric(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
