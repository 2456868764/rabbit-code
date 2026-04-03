package anthropic

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
)

// hashRequestPayloadSHA256Hex reads req.Body (if any), restores Body + GetBody for retries, and returns lowercase hex SHA256.
func hashRequestPayloadSHA256Hex(req *http.Request) (string, error) {
	var body []byte
	var err error
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return "", fmt.Errorf("read body for signing: %w", err)
		}
		_ = req.Body.Close()
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
