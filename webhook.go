package goldsky

import (
	"crypto/subtle"
	"net/http"
)

// WebhookSecretHeader is the literal header Goldsky sends on every webhook
// delivery carrying the shared secret.
const WebhookSecretHeader = "goldsky-webhook-secret"

// VerifyWebhookSecret reports whether the provided secret matches the expected
// secret in constant time. Goldsky deliveries carry the secret verbatim in the
// goldsky-webhook-secret header; Goldsky does not document an HMAC signature
// format, so this helper performs a direct constant-time comparison only.
//
// Pass the raw header value as provided and the secret you stored at webhook
// creation time. A non-empty expected secret is required; an empty expected
// secret always returns false to prevent accidental acceptance of unset
// secrets.
func VerifyWebhookSecret(provided, expected string) bool {
	if expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// VerifyWebhookRequest verifies the goldsky-webhook-secret header on an
// http.Request against the expected secret. It returns true only when the
// header is present and matches in constant time.
func VerifyWebhookRequest(r *http.Request, expected string) bool {
	if r == nil {
		return false
	}
	return VerifyWebhookSecret(r.Header.Get(WebhookSecretHeader), expected)
}
