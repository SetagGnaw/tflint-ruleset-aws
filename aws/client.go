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

// Merge returns merged credentials. A single role in other overrides the final
// role while preserving any preceding roles in the provider's role chain.
func (c Credentials) Merge(other Credentials) Credentials {
	if other.AccessKey != "" {
		c.AccessKey = other.AccessKey
	}
	if other.SecretKey != "" {
		c.SecretKey = other.SecretKey
	}
	if other.Profile != "" {
		c.Profile = other.Profile
	}
	if other.CredsFile != "" {
		c.CredsFile = other.CredsFile
	}
	if other.Region != "" {
		c.Region = other.Region
	}
	c.AssumeRoles = append([]AssumeRole(nil), c.AssumeRoles...)
	if len(other.AssumeRoles) > 1 || len(c.AssumeRoles) == 0 {
		c.AssumeRoles = append([]AssumeRole(nil), other.AssumeRoles...)
	} else if len(other.AssumeRoles) == 1 {
		last := &c.AssumeRoles[len(c.AssumeRoles)-1]
		if other.AssumeRoles[0].RoleARN != "" {
			last.RoleARN = other.AssumeRoles[0].RoleARN
		}
		if other.AssumeRoles[0].SessionName != "" {
			last.SessionName = other.AssumeRoles[0].SessionName
		}
		if other.AssumeRoles[0].ExternalID != "" {
			last.ExternalID = other.AssumeRoles[0].ExternalID
		}
		if other.AssumeRoles[0].Policy != "" {
			last.Policy = other.AssumeRoles[0].Policy
		}
	}
	return c
}
