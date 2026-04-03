package anthropic

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func TestBedrockSigV4Signer_Sign_setsAWS4Authorization(t *testing.T) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"AKIAIOSFODNN7EXAMPLE",
			"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"",
		)),
	)
	if err != nil {
		t.Fatal(err)
	}
	s := NewBedrockSigV4SignerFromAWSConfig(cfg)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://bedrock-runtime.us-east-1.amazonaws.com/model/anthropic.claude-3-haiku-20240307-v1%3A0/invoke-with-response-stream",
		strings.NewReader(`{"max_tokens":1,"messages":[],"anthropic_version":"bedrock-2023-05-31"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := s.Sign(ctx, req); err != nil {
		t.Fatal(err)
	}
	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") {
		t.Fatalf("Authorization=%q", auth)
	}
	if req.Header.Get("X-Amz-Date") == "" {
		t.Fatal("missing X-Amz-Date")
	}
}

func TestBedrockSigV4Signer_skipNoops(t *testing.T) {
	t.Setenv("RABBIT_CODE_SKIP_BEDROCK_AUTH", "1")
	s, err := NewBedrockSigV4Signer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err := s.Sign(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != "" {
		t.Fatal("expected no auth when skip")
	}
}
