package anthropic

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

// ShouldSkipPreconnect mirrors apiPreconnect.ts skip conditions (proxy, mTLS, unix socket, cloud providers).
func ShouldSkipPreconnect() bool {
	if features.UseBedrock() || features.UseVertex() || features.UseFoundry() {
		return true
	}
	if strings.TrimSpace(os.Getenv("RABBIT_CODE_UNIX_SOCKET")) != "" {
		return true
	}
	if mTLSEnvPathsSet("RABBIT_CODE_CLIENT_CERT", "RABBIT_CODE_CLIENT_KEY") {
		return true
	}
	for _, k := range []string{
		"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy",
		"ALL_PROXY", "all_proxy",
	} {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			return true
		}
	}
	return false
}

func mTLSEnvPathsSet(certEnv, keyEnv string) bool {
	return strings.TrimSpace(os.Getenv(certEnv)) != "" ||
		strings.TrimSpace(os.Getenv(keyEnv)) != ""
}

// PreconnectHEAD issues a fire-and-forget HEAD to baseURL (apiPreconnect.ts). Errors are ignored.
func PreconnectHEAD(ctx context.Context, client *http.Client, baseURL string) error {
	if ShouldSkipPreconnect() {
		return nil
	}
	if client == nil {
		client = http.DefaultClient
	}
	u := strings.TrimRight(baseURL, "/")
	if u == "" {
		u = BaseURL(ProviderAnthropic)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return err
	}
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx2)
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	_ = resp.Body.Close()
	return nil
}
