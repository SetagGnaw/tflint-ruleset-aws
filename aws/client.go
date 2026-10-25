package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	awsbase "github.com/hashicorp/aws-sdk-go-base/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
)

// AwsClient is a wrapper of the AWS SDK client.
// This is the real implementation that satisfies the interface.
type AwsClient struct {
	IAM         *iam.Client
	EC2         *ec2.Client
	RDS         *rds.Client
	ElastiCache *elasticache.Client
	ELB         *elasticloadbalancing.Client
	ELBV2       *elasticloadbalancingv2.Client
	ECS         *ecs.Client
}

var _ Client = (*AwsClient)(nil)

// Credentials is credentials for AWS used in deep check mode
type Credentials struct {
	AccessKey   string
	SecretKey   string
	Profile     string
	CredsFile   string
	AssumeRoles []AssumeRole
	Region      string
}

// NewClient returns a new Client with configured session
func NewClient(creds Credentials) (Client, error) {
	logger.Info("Initialize AWS Client")

	config, err := getBaseConfig(creds)
	if err != nil {
		return nil, err
	}

	_, awsConfig, diags := awsbase.GetAwsConfig(context.Background(), config)
	for _, diag := range diags.Errors() {
		err = errors.Join(err, fmt.Errorf("%s; %s", diag.Summary(), diag.Detail()))
	}
	if err != nil {
		return nil, err
	}

	return &AwsClient{
		IAM:         iam.NewFromConfig(awsConfig),
		EC2:         ec2.NewFromConfig(awsConfig),
		RDS:         rds.NewFromConfig(awsConfig),
		ElastiCache: elasticache.NewFromConfig(awsConfig),
		ELB:         elasticloadbalancing.NewFromConfig(awsConfig),
		ELBV2:       elasticloadbalancingv2.NewFromConfig(awsConfig),
		ECS:         ecs.NewFromConfig(awsConfig),
	}, nil
}

func getBaseConfig(creds Credentials) (*awsbase.Config, error) {
	expandedCredsFile, err := homedir.Expand(creds.CredsFile)
	if err != nil {
		return nil, err
	}

	config := &awsbase.Config{
		AccessKey:              creds.AccessKey,
		SecretKey:              creds.SecretKey,
		Profile:                creds.Profile,
		Region:                 creds.Region,
		CallerName:             "tflint-ruleset-aws",
		CallerDocumentationURL: "https://github.com/terraform-linters/tflint-ruleset-aws/blob/master/docs/deep_checking.md",
	}

	for _, assumeRole := range creds.AssumeRoles {
		if assumeRole.RoleARN == "" && assumeRole.ExternalID == "" && assumeRole.Policy == "" && assumeRole.SessionName == "" {
			continue
		}
		config.AssumeRole = append(config.AssumeRole, awsbase.AssumeRole{
			RoleARN:     assumeRole.RoleARN,
			ExternalID:  assumeRole.ExternalID,
			Policy:      assumeRole.Policy,
			SessionName: assumeRole.SessionName,
		})
	}
	if expandedCredsFile != "" {
		config.SharedCredentialsFiles = []string{expandedCredsFile}
	}

	return config, nil
}

// Merge credentials from the TFLint plugin configuration (overrides) to
// credentials read from the Terraform provider "aws" block (base).
// Merge returns a new Credentials where non-empty fields from overrides
// take precedence over base. For AssumeRoles, the merge strategy depends
// on the number of roles in overrides:
//   - 0 roles: base's chain is kept as-is.
//   - 1 role:  patches base's final role field-by-field.
//   - >1 roles: replaces base's chain entirely with overrides' chain.
func (base Credentials) Merge(overrides Credentials) Credentials {
	if overrides.AccessKey != "" {
		base.AccessKey = overrides.AccessKey
	}
	if overrides.SecretKey != "" {
		base.SecretKey = overrides.SecretKey
	}
	if overrides.Profile != "" {
		base.Profile = overrides.Profile
	}
	if overrides.CredsFile != "" {
		base.CredsFile = overrides.CredsFile
	}
	if overrides.Region != "" {
		base.Region = overrides.Region
	}

	base.AssumeRoles = mergeAssumeRoles(base.AssumeRoles, overrides.AssumeRoles)
	return base
}

// mergeAssumeRoles combines two assume-role slices. A defensive copy is
// always made so the caller's original slices are never mutated.
func mergeAssumeRoles(base, overrides []AssumeRole) []AssumeRole {
	base = append([]AssumeRole(nil), base...)

	switch {
	// No overrides — keep base's chain unchanged.
	case len(overrides) == 0:
		return base

	// Overrides specify a full chain (>1 role) or base has no chain to
	// preserve — replace entirely with the overrides' chain.
	case len(overrides) > 1 || len(base) == 0:
		return append([]AssumeRole(nil), overrides...)

	// Single override role — patch non-empty fields into base's final role while
	// preserving any preceding roles in the provider chain.
	case len(overrides) == 1:
		patchAssumeRole(&base[len(base)-1], overrides[0])
	}

	return base
}

// patchAssumeRole overwrites non-empty fields in dst with values from src.
func patchAssumeRole(dst *AssumeRole, src AssumeRole) {
	if src.RoleARN != "" {
		dst.RoleARN = src.RoleARN
	}
	if src.SessionName != "" {
		dst.SessionName = src.SessionName
	}
	if src.ExternalID != "" {
		dst.ExternalID = src.ExternalID
	}
	if src.Policy != "" {
		dst.Policy = src.Policy
	}
}
