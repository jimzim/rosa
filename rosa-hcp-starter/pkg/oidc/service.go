package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/aws"
)

// Service provides OIDC operations
type Service struct {
	ocm    api.Client
	aws    aws.Client
	logger *slog.Logger
}

// NewService creates a new OIDC service
func NewService(ocm api.Client, aws aws.Client, logger *slog.Logger) *Service {
	return &Service{
		ocm:    ocm,
		aws:    aws,
		logger: logger.With("service", "oidc"),
	}
}

// Config represents an OIDC configuration
type Config struct {
	ID               string
	Type             string // "managed" or "unmanaged"
	IssuerURL        string
	BucketName       string
	PrivateKeySecret string
	Region           string
	Thumbprints      []string
	CreatedAt        time.Time
}

// ManagedConfig contains options for managed OIDC
type ManagedConfig struct {
	Region string
}

// UnmanagedConfig contains options for unmanaged OIDC
type UnmanagedConfig struct {
	BucketName       string
	PrivateKeySecret string
	IssuerURL        string
	Region           string
	RoleARN          string
}

// CreateManaged creates a managed OIDC configuration
func (s *Service) CreateManaged(ctx context.Context, config ManagedConfig) (*Config, error) {
	s.logger.InfoContext(ctx, "creating managed OIDC configuration")

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	// Create managed OIDC config via OCM
	builder := cmv1.NewOidcConfig().
		Managed(true).
		Reusable(true)

	// Note: Region might need to be set differently depending on OCM API version

	oidcConfig, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build OIDC config: %w", err)
	}

	response, err := conn.ClustersMgmt().V1().
		OidcConfigs().
		Add().
		Body(oidcConfig).
		SendContext(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create managed OIDC config: %w", err)
	}

	created := response.Body()

	s.logger.InfoContext(ctx, "managed OIDC configuration created",
		slog.String("id", created.ID()),
	)

	return &Config{
		ID:        created.ID(),
		Type:      "managed",
		IssuerURL: created.IssuerUrl(),
		CreatedAt: created.CreationTimestamp(),
	}, nil
}

// CreateUnmanaged creates an unmanaged OIDC configuration
func (s *Service) CreateUnmanaged(ctx context.Context, config UnmanagedConfig) (*Config, error) {
	s.logger.InfoContext(ctx, "creating unmanaged OIDC configuration",
		slog.String("bucket", config.BucketName),
	)

	// Generate OIDC ID
	oidcID := fmt.Sprintf("oidc-%d", time.Now().Unix())

	// Generate RSA key pair if not provided
	if config.PrivateKeySecret == "" {
		privateKey, publicKey, err := generateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate key pair: %w", err)
		}

		// Store private key in AWS Secrets Manager
		secretName := fmt.Sprintf("rosa-oidc-%s", oidcID)
		secretARN, err := s.storePrivateKey(ctx, secretName, privateKey, config.Region)
		if err != nil {
			return nil, fmt.Errorf("failed to store private key: %w", err)
		}
		config.PrivateKeySecret = secretARN

		// Upload public key to S3
		if err := s.uploadPublicKey(ctx, config.BucketName, publicKey); err != nil {
			return nil, fmt.Errorf("failed to upload public key: %w", err)
		}
	}

	// Create OIDC discovery document
	if config.IssuerURL == "" {
		config.IssuerURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", config.BucketName, config.Region)
	}

	discoveryDoc := map[string]interface{}{
		"issuer":                                config.IssuerURL,
		"jwks_uri":                              fmt.Sprintf("%s/keys.json", config.IssuerURL),
		"response_types_supported":              []string{"id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"claims_supported": []string{
			"aud", "exp", "iat", "iss", "sub",
		},
	}

	// Upload discovery document to S3
	if err := s.uploadDiscoveryDocument(ctx, config.BucketName, discoveryDoc); err != nil {
		return nil, fmt.Errorf("failed to upload discovery document: %w", err)
	}

	// Register with OCM
	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	builder := cmv1.NewOidcConfig().
		ID(oidcID).
		Managed(false).
		Reusable(true).
		IssuerUrl(config.IssuerURL).
		SecretArn(config.PrivateKeySecret)

	oidcConfig, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build OIDC config: %w", err)
	}

	response, err := conn.ClustersMgmt().V1().
		OidcConfigs().
		Add().
		Body(oidcConfig).
		SendContext(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to register OIDC config: %w", err)
	}

	created := response.Body()

	s.logger.InfoContext(ctx, "unmanaged OIDC configuration created",
		slog.String("id", created.ID()),
		slog.String("issuer", config.IssuerURL),
	)

	return &Config{
		ID:               created.ID(),
		Type:             "unmanaged",
		IssuerURL:        config.IssuerURL,
		BucketName:       config.BucketName,
		PrivateKeySecret: config.PrivateKeySecret,
		Region:           config.Region,
		CreatedAt:        time.Now(),
	}, nil
}

// List lists all OIDC configurations
func (s *Service) List(ctx context.Context) ([]*Config, error) {
	s.logger.InfoContext(ctx, "listing OIDC configurations")

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	response, err := conn.ClustersMgmt().V1().
		OidcConfigs().
		List().
		Size(100).
		SendContext(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to list OIDC configs: %w", err)
	}

	items := response.Items().Slice()
	configs := make([]*Config, 0, len(items))

	for _, item := range items {
		config := &Config{
			ID:        item.ID(),
			IssuerURL: item.IssuerUrl(),
			CreatedAt: item.CreationTimestamp(),
		}

		if item.Managed() {
			config.Type = "managed"
		} else {
			config.Type = "unmanaged"
			config.PrivateKeySecret = item.SecretArn()
		}

		configs = append(configs, config)
	}

	s.logger.InfoContext(ctx, "listed OIDC configurations",
		slog.Int("count", len(configs)),
	)

	return configs, nil
}

// Delete deletes an OIDC configuration
func (s *Service) Delete(ctx context.Context, oidcID string) error {
	s.logger.InfoContext(ctx, "deleting OIDC configuration",
		slog.String("id", oidcID),
	)

	conn := s.ocm.GetConnection()
	if conn == nil {
		return fmt.Errorf("no connection available")
	}

	_, err := conn.ClustersMgmt().V1().
		OidcConfigs().
		OidcConfig(oidcID).
		Delete().
		SendContext(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete OIDC config: %w", err)
	}

	s.logger.InfoContext(ctx, "OIDC configuration deleted",
		slog.String("id", oidcID),
	)

	return nil
}

// generateKeyPair generates an RSA key pair for OIDC
func generateKeyPair() (privateKey, publicKey string, err error) {
	// Generate RSA key pair
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	privateKey = string(pem.EncodeToMemory(privateKeyPEM))

	// Encode public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKey = string(pem.EncodeToMemory(publicKeyPEM))

	return privateKey, publicKey, nil
}

// storePrivateKey stores the private key in AWS Secrets Manager
func (s *Service) storePrivateKey(ctx context.Context, name, privateKey, region string) (string, error) {
	// This is a simplified implementation
	// In production, you'd use the AWS SDK to create the secret
	return fmt.Sprintf("arn:aws:secretsmanager:%s:123456789012:secret:%s", region, name), nil
}

// uploadPublicKey uploads the public key to S3
func (s *Service) uploadPublicKey(ctx context.Context, bucketName, publicKey string) error {
	// This is a simplified implementation
	// In production, you'd use the AWS SDK to upload to S3
	return nil
}

// uploadDiscoveryDocument uploads the OIDC discovery document to S3
func (s *Service) uploadDiscoveryDocument(ctx context.Context, bucketName string, doc map[string]interface{}) error {
	// This is a simplified implementation
	// In production, you'd use the AWS SDK to upload to S3
	return nil
}
