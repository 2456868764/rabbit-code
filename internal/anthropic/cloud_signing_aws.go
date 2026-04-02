package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
)

const bedrockRuntimeService = "bedrock-runtime"

// BedrockSigV4Signer signs outgoing Bedrock Runtime HTTP requests with AWS Signature Version 4 (AC4-6).
type BedrockSigV4Signer struct {
	region string
	creds  aws.CredentialsProvider
	signer *v4.Signer
	skip   bool
}

// NewBedrockSigV4Signer loads default AWS config and credentials (env, shared config, SSO, IMDS, etc.).
func NewBedrockSigV4Signer(ctx context.Context) (*BedrockSigV4Signer, error) {
	if features.SkipBedrockAuth() {
		return &BedrockSigV4Signer{skip: true}, nil
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("bedrock signing: load aws config: %w", err)
	}
	return newBedrockSigV4SignerFromAWSConfig(cfg), nil
}

// NewBedrockSigV4SignerFromAWSConfig builds a signer from an existing aws.Config (tests, custom chains).
func NewBedrockSigV4SignerFromAWSConfig(cfg aws.Config) *BedrockSigV4Signer {
	return newBedrockSigV4SignerFromAWSConfig(cfg)
}

func newBedrockSigV4SignerFromAWSConfig(cfg aws.Config) *BedrockSigV4Signer {
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_REGION"))
		if region == "" {
			region = strings.TrimSpace(os.Getenv("AWS_DEFAULT_REGION"))
		}
		if region == "" {
			region = "us-east-1"
		}
	}
	return &BedrockSigV4Signer{
		region: region,
		creds:  cfg.Credentials,
		signer: v4.NewSigner(),
	}
}

// Sign implements CloudRequestSigner. It sets SigV4 auth headers and x-amz-content-sha256.
func (s *BedrockSigV4Signer) Sign(ctx context.Context, req *http.Request) error {
	if s == nil || s.skip {
		return nil
	}
	creds, err := s.creds.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("bedrock signing: retrieve credentials: %w", err)
	}
	payloadHash, err := hashRequestPayloadSHA256Hex(req)
	if err != nil {
		return err
	}
	if err := s.signer.SignHTTP(ctx, creds, req, payloadHash, bedrockRuntimeService, s.region, time.Now()); err != nil {
		return fmt.Errorf("bedrock signing: %w", err)
	}
	return nil
}
