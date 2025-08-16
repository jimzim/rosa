package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// HCP Account Roles - only 3 for HCP (no control plane role)
const (
	HCPInstallerRole = "Installer"
	HCPSupportRole   = "Support"
	HCPWorkerRole    = "Worker"
	
	DefaultPrefix = "ManagedOpenShift-HCP-ROSA"
)

// RoleConfig holds configuration for role creation
type RoleConfig struct {
	Prefix              string
	AccountID           string
	Region              string
	PermissionsBoundary string
	Path                string
	Tags                map[string]string
}

// RoleInfo contains information about a created role
type RoleInfo struct {
	RoleName string
	RoleARN  string
	RoleType string
	Policies []string
}

// Service provides IAM operations
type Service interface {
	CreateAccountRoles(ctx context.Context, config RoleConfig) ([]RoleInfo, error)
	CreateOperatorRoles(ctx context.Context, config RoleConfig, oidcEndpoint string, clusterID string) ([]RoleInfo, error)
	DeleteAccountRoles(ctx context.Context, prefix string) error
	DeleteOperatorRoles(ctx context.Context, prefix string, clusterID string) error
	ListAccountRoles(ctx context.Context, prefix string) ([]RoleInfo, error)
	ValidateAccountRoles(ctx context.Context, prefix string) error
	GetCallerIdentity(ctx context.Context) (*CallerIdentity, error)
}

// CallerIdentity represents AWS caller identity
type CallerIdentity struct {
	AccountID string
	UserID    string
	ARN       string
}

type service struct {
	iamClient *iam.Client
	stsClient *sts.Client
	logger    *slog.Logger
	partition string
}

// NewService creates a new IAM service
func NewService(ctx context.Context, region string, logger *slog.Logger) (Service, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Determine partition from region
	partition := "aws"
	if strings.HasPrefix(region, "us-gov-") {
		partition = "aws-us-gov"
	} else if strings.HasPrefix(region, "cn-") {
		partition = "aws-cn"
	}

	return &service{
		iamClient: iam.NewFromConfig(cfg),
		stsClient: sts.NewFromConfig(cfg),
		logger:    logger,
		partition: partition,
	}, nil
}

// GetCallerIdentity returns the AWS caller identity
func (s *service) GetCallerIdentity(ctx context.Context) (*CallerIdentity, error) {
	result, err := s.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity: %w", err)
	}

	return &CallerIdentity{
		AccountID: *result.Account,
		UserID:    *result.UserId,
		ARN:       *result.Arn,
	}, nil
}

// CreateAccountRoles creates the HCP account roles
func (s *service) CreateAccountRoles(ctx context.Context, cfg RoleConfig) ([]RoleInfo, error) {
	s.logger.Info("Creating HCP account roles", "prefix", cfg.Prefix)

	if cfg.AccountID == "" {
		identity, err := s.GetCallerIdentity(ctx)
		if err != nil {
			return nil, err
		}
		cfg.AccountID = identity.AccountID
	}

	var roles []RoleInfo

	// Create the three HCP roles
	roleTypes := []string{HCPInstallerRole, HCPSupportRole, HCPWorkerRole}
	
	for _, roleType := range roleTypes {
		roleName := fmt.Sprintf("%s-%s", cfg.Prefix, roleType)
		
		// Create trust policy
		trustPolicy := s.getHCPTrustPolicy(cfg.AccountID, roleType)
		
		// Create or update role
		s.logger.Info("Creating role", "name", roleName, "type", roleType)
		roleARN, err := s.ensureRole(ctx, roleName, trustPolicy, cfg)
		if err != nil {
			return roles, fmt.Errorf("failed to create role %s: %w", roleName, err)
		}

		// Attach policies
		policies, err := s.attachHCPPolicies(ctx, roleName, roleType, cfg)
		if err != nil {
			return roles, fmt.Errorf("failed to attach policies to %s: %w", roleName, err)
		}

		roles = append(roles, RoleInfo{
			RoleName: roleName,
			RoleARN:  roleARN,
			RoleType: roleType,
			Policies: policies,
		})

		s.logger.Info("Created role successfully", "role", roleName, "arn", roleARN)
	}

	return roles, nil
}

// ensureRole creates or updates an IAM role
func (s *service) ensureRole(ctx context.Context, roleName string, trustPolicy string, cfg RoleConfig) (string, error) {
	// Check if role exists
	existingRole, err := s.iamClient.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	
	if err != nil {
		// Role doesn't exist, create it
		createInput := &iam.CreateRoleInput{
			RoleName:                 aws.String(roleName),
			AssumeRolePolicyDocument: aws.String(trustPolicy),
			Description:              aws.String(fmt.Sprintf("ROSA HCP %s role", roleName)),
			Tags:                     s.buildTags(cfg.Tags),
		}
		
		if cfg.Path != "" {
			createInput.Path = aws.String(cfg.Path)
		}
		
		if cfg.PermissionsBoundary != "" {
			createInput.PermissionsBoundary = aws.String(cfg.PermissionsBoundary)
		}
		
		result, err := s.iamClient.CreateRole(ctx, createInput)
		if err != nil {
			return "", fmt.Errorf("failed to create role: %w", err)
		}
		
		return *result.Role.Arn, nil
	}
	
	// Role exists, update trust policy
	_, err = s.iamClient.UpdateAssumeRolePolicy(ctx, &iam.UpdateAssumeRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyDocument: aws.String(trustPolicy),
	})
	if err != nil {
		return "", fmt.Errorf("failed to update trust policy: %w", err)
	}
	
	// Update tags
	if len(cfg.Tags) > 0 {
		_, err = s.iamClient.TagRole(ctx, &iam.TagRoleInput{
			RoleName: aws.String(roleName),
			Tags:     s.buildTags(cfg.Tags),
		})
		if err != nil {
			s.logger.Warn("Failed to update role tags", "error", err)
		}
	}
	
	return *existingRole.Role.Arn, nil
}

// getHCPTrustPolicy returns the trust policy for HCP roles
func (s *service) getHCPTrustPolicy(accountID string, roleType string) string {
	var principal string
	
	switch roleType {
	case HCPInstallerRole, HCPSupportRole:
		// These roles trust the Red Hat installer/support account
		principal = fmt.Sprintf("arn:%s:iam::710019948333:role/RH-Managed-OpenShift-Installer", s.partition)
		if roleType == HCPSupportRole {
			principal = fmt.Sprintf("arn:%s:iam::710019948333:role/RH-Technical-Support-Access", s.partition)
		}
		
		policy := map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"AWS": principal,
					},
					"Action": "sts:AssumeRole",
				},
			},
		}
		
		policyJSON, _ := json.Marshal(policy)
		return string(policyJSON)
		
	case HCPWorkerRole:
		// Worker role trusts EC2 service
		policy := map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": "ec2.amazonaws.com",
					},
					"Action": "sts:AssumeRole",
				},
			},
		}
		
		policyJSON, _ := json.Marshal(policy)
		return string(policyJSON)
		
	default:
		return ""
	}
}

// attachHCPPolicies attaches the required policies to HCP roles
func (s *service) attachHCPPolicies(ctx context.Context, roleName string, roleType string, cfg RoleConfig) ([]string, error) {
	var policies []string
	
	// For now, we'll attach AWS managed policies
	// In production, these should be custom policies with minimal permissions
	managedPolicies := s.getHCPManagedPolicies(roleType)
	
	for _, policyARN := range managedPolicies {
		_, err := s.iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
			RoleName:  aws.String(roleName),
			PolicyArn: aws.String(policyARN),
		})
		if err != nil {
			// Check if already attached
			if !strings.Contains(err.Error(), "Duplicate") {
				return policies, fmt.Errorf("failed to attach policy %s: %w", policyARN, err)
			}
		}
		policies = append(policies, policyARN)
	}
	
	// Create and attach inline policies for specific permissions
	inlinePolicy := s.getHCPInlinePolicy(roleType)
	if inlinePolicy != "" {
		policyName := fmt.Sprintf("%s-Policy", roleType)
		_, err := s.iamClient.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
			RoleName:       aws.String(roleName),
			PolicyName:     aws.String(policyName),
			PolicyDocument: aws.String(inlinePolicy),
		})
		if err != nil {
			return policies, fmt.Errorf("failed to attach inline policy: %w", err)
		}
		policies = append(policies, fmt.Sprintf("inline:%s", policyName))
	}
	
	return policies, nil
}

// getHCPManagedPolicies returns AWS managed policies for HCP roles
func (s *service) getHCPManagedPolicies(roleType string) []string {
	switch roleType {
	case HCPInstallerRole:
		// Installer needs permissions to create cluster resources
		return []string{
			fmt.Sprintf("arn:%s:iam::aws:policy/PowerUserAccess", s.partition),
		}
	case HCPSupportRole:
		// Support needs read-only access
		return []string{
			fmt.Sprintf("arn:%s:iam::aws:policy/ReadOnlyAccess", s.partition),
		}
	case HCPWorkerRole:
		// Worker nodes need specific EC2 permissions
		return []string{
			fmt.Sprintf("arn:%s:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly", s.partition),
			fmt.Sprintf("arn:%s:iam::aws:policy/AmazonEC2FullAccess", s.partition),
		}
	default:
		return []string{}
	}
}

// getHCPInlinePolicy returns inline policy documents for HCP roles
func (s *service) getHCPInlinePolicy(roleType string) string {
	// These would be more restrictive in production
	// For now, returning empty to rely on managed policies
	return ""
}

// CreateOperatorRoles creates operator roles for a cluster
func (s *service) CreateOperatorRoles(ctx context.Context, cfg RoleConfig, oidcEndpoint string, clusterID string) ([]RoleInfo, error) {
	s.logger.Info("Creating operator roles", "cluster", clusterID, "oidc", oidcEndpoint)
	
	if oidcEndpoint == "" {
		return nil, fmt.Errorf("OIDC endpoint is required for operator roles")
	}
	
	if clusterID == "" {
		return nil, fmt.Errorf("cluster ID is required for operator roles")
	}
	
	// Clean OIDC endpoint (remove https:// if present)
	oidcEndpoint = strings.TrimPrefix(oidcEndpoint, "https://")
	
	var roles []RoleInfo
	
	// Define HCP operator roles
	operators := []struct {
		name      string
		namespace string
		policy    string
	}{
		{"ingress-operator", "openshift-ingress-operator", "IngressOperator"},
		{"cluster-csi-drivers", "openshift-cluster-csi-drivers", "CSIDriver"},
		{"cloud-network-config-controller", "openshift-cloud-network-config-controller", "CloudNetworkConfig"},
		{"kube-controller-manager", "kube-system", "KubeControllerManager"},
		{"kms-provider", "kube-system", "KMSProvider"},
		{"control-plane-operator", "openshift-control-plane-operator", "ControlPlaneOperator"},
		{"image-registry-operator", "openshift-image-registry", "ImageRegistry"},
	}
	
	for _, op := range operators {
		roleName := fmt.Sprintf("%s-%s-%s", cfg.Prefix, clusterID, op.name)
		// Truncate if too long (IAM limit is 64 chars)
		if len(roleName) > 64 {
			roleName = roleName[:64]
		}
		
		// Create trust policy for operator
		trustPolicy := s.getOperatorTrustPolicy(oidcEndpoint, op.namespace, op.name, cfg.AccountID)
		
		s.logger.Info("Creating operator role", "name", roleName, "operator", op.name)
		roleARN, err := s.ensureRole(ctx, roleName, trustPolicy, cfg)
		if err != nil {
			return roles, fmt.Errorf("failed to create operator role %s: %w", roleName, err)
		}
		
		// Attach operator-specific policies
		policies, err := s.attachOperatorPolicies(ctx, roleName, op.policy, cfg)
		if err != nil {
			return roles, fmt.Errorf("failed to attach policies to %s: %w", roleName, err)
		}
		
		roles = append(roles, RoleInfo{
			RoleName: roleName,
			RoleARN:  roleARN,
			RoleType: fmt.Sprintf("Operator:%s", op.name),
			Policies: policies,
		})
		
		s.logger.Info("Created operator role successfully", "role", roleName, "arn", roleARN)
	}
	
	return roles, nil
}

// DeleteAccountRoles deletes account roles with the given prefix
func (s *service) DeleteAccountRoles(ctx context.Context, prefix string) error {
	s.logger.Info("Deleting account roles", "prefix", prefix)
	
	roleTypes := []string{HCPInstallerRole, HCPSupportRole, HCPWorkerRole}
	
	for _, roleType := range roleTypes {
		roleName := fmt.Sprintf("%s-%s", prefix, roleType)
		
		// Detach policies first
		policies, err := s.iamClient.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			if strings.Contains(err.Error(), "NoSuchEntity") {
				continue
			}
			return fmt.Errorf("failed to list policies for %s: %w", roleName, err)
		}
		
		for _, policy := range policies.AttachedPolicies {
			_, err = s.iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				RoleName:  aws.String(roleName),
				PolicyArn: policy.PolicyArn,
			})
			if err != nil {
				s.logger.Warn("Failed to detach policy", "role", roleName, "policy", *policy.PolicyArn, "error", err)
			}
		}
		
		// Delete inline policies
		inlinePolicies, err := s.iamClient.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
			RoleName: aws.String(roleName),
		})
		if err == nil {
			for _, policyName := range inlinePolicies.PolicyNames {
				_, err = s.iamClient.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
					RoleName:   aws.String(roleName),
					PolicyName: aws.String(policyName),
				})
				if err != nil {
					s.logger.Warn("Failed to delete inline policy", "role", roleName, "policy", policyName, "error", err)
				}
			}
		}
		
		// Delete the role
		_, err = s.iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "NoSuchEntity") {
				return fmt.Errorf("failed to delete role %s: %w", roleName, err)
			}
		}
		
		s.logger.Info("Deleted role", "name", roleName)
	}
	
	return nil
}

// getOperatorTrustPolicy returns the trust policy for operator roles
func (s *service) getOperatorTrustPolicy(oidcEndpoint string, namespace string, operatorName string, accountID string) string {
	// Create trust policy that trusts the OIDC provider for specific service accounts
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Federated": fmt.Sprintf("arn:%s:iam::%s:oidc-provider/%s", s.partition, accountID, oidcEndpoint),
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": map[string]interface{}{
					"StringEquals": map[string]string{
						fmt.Sprintf("%s:sub", oidcEndpoint): fmt.Sprintf("system:serviceaccount:%s:%s", namespace, operatorName),
					},
				},
			},
		},
	}
	
	policyJSON, _ := json.Marshal(policy)
	return string(policyJSON)
}

// attachOperatorPolicies attaches policies to operator roles
func (s *service) attachOperatorPolicies(ctx context.Context, roleName string, policyType string, cfg RoleConfig) ([]string, error) {
	var policies []string
	
	// Create inline policy with operator-specific permissions
	// In production, these would be fine-tuned for each operator's needs
	inlinePolicy := s.getOperatorInlinePolicy(policyType)
	if inlinePolicy != "" {
		policyName := fmt.Sprintf("%s-Policy", policyType)
		_, err := s.iamClient.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
			RoleName:       aws.String(roleName),
			PolicyName:     aws.String(policyName),
			PolicyDocument: aws.String(inlinePolicy),
		})
		if err != nil {
			return policies, fmt.Errorf("failed to attach inline policy: %w", err)
		}
		policies = append(policies, fmt.Sprintf("inline:%s", policyName))
	}
	
	return policies, nil
}

// getOperatorInlinePolicy returns inline policy for specific operators
func (s *service) getOperatorInlinePolicy(policyType string) string {
	// These are simplified policies - production would need more specific permissions
	switch policyType {
	case "IngressOperator":
		policy := map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Action": []string{
						"elasticloadbalancing:*",
						"ec2:DescribeVpcs",
						"ec2:DescribeSubnets",
						"ec2:DescribeSecurityGroups",
						"route53:ListHostedZones",
						"route53:ChangeResourceRecordSets",
						"tag:GetResources",
					},
					"Resource": "*",
				},
			},
		}
		policyJSON, _ := json.Marshal(policy)
		return string(policyJSON)
		
	case "CSIDriver":
		policy := map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Action": []string{
						"ec2:CreateVolume",
						"ec2:DeleteVolume",
						"ec2:AttachVolume",
						"ec2:DetachVolume",
						"ec2:DescribeVolumes",
						"ec2:DescribeInstances",
						"ec2:CreateSnapshot",
						"ec2:DeleteSnapshot",
						"ec2:DescribeSnapshots",
						"ec2:CreateTags",
					},
					"Resource": "*",
				},
			},
		}
		policyJSON, _ := json.Marshal(policy)
		return string(policyJSON)
		
	case "ImageRegistry":
		policy := map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Action": []string{
						"s3:CreateBucket",
						"s3:DeleteBucket",
						"s3:GetBucketLocation",
						"s3:ListBucket",
						"s3:PutObject",
						"s3:GetObject",
						"s3:DeleteObject",
						"s3:PutBucketTagging",
						"s3:GetBucketTagging",
						"s3:PutBucketPublicAccessBlock",
						"s3:GetBucketPublicAccessBlock",
						"s3:PutEncryptionConfiguration",
						"s3:GetEncryptionConfiguration",
					},
					"Resource": "*",
				},
			},
		}
		policyJSON, _ := json.Marshal(policy)
		return string(policyJSON)
		
	default:
		// Generic policy for other operators
		policy := map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Action": []string{
						"ec2:Describe*",
						"iam:GetRole",
						"iam:ListRolePolicies",
						"tag:GetResources",
					},
					"Resource": "*",
				},
			},
		}
		policyJSON, _ := json.Marshal(policy)
		return string(policyJSON)
	}
}

// DeleteOperatorRoles deletes operator roles for a cluster
func (s *service) DeleteOperatorRoles(ctx context.Context, prefix string, clusterID string) error {
	s.logger.Info("Deleting operator roles", "prefix", prefix, "cluster", clusterID)
	
	// List all roles with the pattern prefix-clusterID-*
	rolePrefix := fmt.Sprintf("%s-%s-", prefix, clusterID)
	
	paginator := iam.NewListRolesPaginator(s.iamClient, &iam.ListRolesInput{
		PathPrefix: aws.String("/"),
	})
	
	var rolesToDelete []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list roles: %w", err)
		}
		
		for _, role := range page.Roles {
			if strings.HasPrefix(*role.RoleName, rolePrefix) {
				rolesToDelete = append(rolesToDelete, *role.RoleName)
			}
		}
	}
	
	// Delete each role
	for _, roleName := range rolesToDelete {
		// Detach policies first
		policies, err := s.iamClient.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			s.logger.Warn("Failed to list policies", "role", roleName, "error", err)
			continue
		}
		
		for _, policy := range policies.AttachedPolicies {
			_, err = s.iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				RoleName:  aws.String(roleName),
				PolicyArn: policy.PolicyArn,
			})
			if err != nil {
				s.logger.Warn("Failed to detach policy", "role", roleName, "policy", *policy.PolicyArn, "error", err)
			}
		}
		
		// Delete inline policies
		inlinePolicies, err := s.iamClient.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
			RoleName: aws.String(roleName),
		})
		if err == nil {
			for _, policyName := range inlinePolicies.PolicyNames {
				_, err = s.iamClient.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
					RoleName:   aws.String(roleName),
					PolicyName: aws.String(policyName),
				})
				if err != nil {
					s.logger.Warn("Failed to delete inline policy", "role", roleName, "policy", policyName, "error", err)
				}
			}
		}
		
		// Delete the role
		_, err = s.iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			s.logger.Warn("Failed to delete role", "role", roleName, "error", err)
		} else {
			s.logger.Info("Deleted operator role", "name", roleName)
		}
	}
	
	return nil
}

// ListAccountRoles lists account roles with the given prefix
func (s *service) ListAccountRoles(ctx context.Context, prefix string) ([]RoleInfo, error) {
	var roles []RoleInfo
	
	roleTypes := []string{HCPInstallerRole, HCPSupportRole, HCPWorkerRole}
	
	for _, roleType := range roleTypes {
		roleName := fmt.Sprintf("%s-%s", prefix, roleType)
		
		role, err := s.iamClient.GetRole(ctx, &iam.GetRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			if strings.Contains(err.Error(), "NoSuchEntity") {
				continue
			}
			return roles, fmt.Errorf("failed to get role %s: %w", roleName, err)
		}
		
		// Get attached policies
		policies, err := s.iamClient.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			return roles, fmt.Errorf("failed to list policies for %s: %w", roleName, err)
		}
		
		var policyList []string
		for _, policy := range policies.AttachedPolicies {
			policyList = append(policyList, *policy.PolicyArn)
		}
		
		roles = append(roles, RoleInfo{
			RoleName: roleName,
			RoleARN:  *role.Role.Arn,
			RoleType: roleType,
			Policies: policyList,
		})
	}
	
	return roles, nil
}

// ValidateAccountRoles validates that all required account roles exist
func (s *service) ValidateAccountRoles(ctx context.Context, prefix string) error {
	roleTypes := []string{HCPInstallerRole, HCPSupportRole, HCPWorkerRole}
	
	for _, roleType := range roleTypes {
		roleName := fmt.Sprintf("%s-%s", prefix, roleType)
		
		_, err := s.iamClient.GetRole(ctx, &iam.GetRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			if strings.Contains(err.Error(), "NoSuchEntity") {
				return fmt.Errorf("role %s does not exist", roleName)
			}
			return fmt.Errorf("failed to validate role %s: %w", roleName, err)
		}
	}
	
	s.logger.Info("All account roles validated successfully", "prefix", prefix)
	return nil
}

// buildTags converts a map to IAM tags
func (s *service) buildTags(tags map[string]string) []types.Tag {
	var iamTags []types.Tag
	for k, v := range tags {
		iamTags = append(iamTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	// Add default tags
	iamTags = append(iamTags,
		types.Tag{
			Key:   aws.String("rosa_hcp"),
			Value: aws.String("true"),
		},
		types.Tag{
			Key:   aws.String("managed"),
			Value: aws.String("true"),
		},
		types.Tag{
			Key:   aws.String("service"),
			Value: aws.String("ROSA"),
		},
	)
	return iamTags
}
