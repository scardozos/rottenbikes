package httpserver

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Dependency metrics for the sign-up / login path. The generic HTTP metrics
// cannot tell a rejected captcha from an unreachable hCaptcha, or show that
// emails stopped going out (e.g. an expired Mailtrap token), so these are
// what alerts should be built on.
var (
	// result: success | failure (hCaptcha rejected the token) | error
	// (hCaptcha unreachable / bad response) | not_configured | skipped (dev).
	captchaVerificationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "captcha_verifications_total",
			Help: "hCaptcha verifications by result",
		},
		[]string{"result"},
	)

	// kind: register | magic_link; result: success | failure.
	emailsSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "emails_sent_total",
			Help: "Transactional emails by sender, kind and result",
		},
		[]string{"sender", "kind", "result"},
	)
)
