package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"skeji/pkg/logger"
	"strings"
)

func WhatsAppSignatureVerification(appSecret string, log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			signature := extractSignature(r)

			if signature == "" {
				logAndReject(w, log, r, "Missing X-Hub-Signature-256 header")
				return
			}

			body, err := readAndRestoreBody(r)
			if err != nil {
				logAndReject(w, log, r, "Failed to read request body")
				return
			}

			if !verifySignature(body, signature, appSecret) {
				logAndReject(w, log, r, "Invalid webhook signature")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractSignature(r *http.Request) string {
	header := r.Header.Get("X-Hub-Signature-256")
	if header == "" {
		return ""
	}

	signature, found := strings.CutPrefix(header, "sha256=")
	if found {
		return signature
	}

	return header
}

func readAndRestoreBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return body, nil
}

func verifySignature(body []byte, receivedSignature string, appSecret string) bool {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedSignature), []byte(receivedSignature))
}

func logAndReject(w http.ResponseWriter, log *logger.Logger, r *http.Request, reason string) {
	requestID := ""
	if rid := r.Context().Value(RequestIDKey); rid != nil {
		requestID = rid.(string)
	}

	log.Warn("WhatsApp webhook verification failed",
		"request_id", requestID,
		"reason", reason,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"Unauthorized"}`))
}
