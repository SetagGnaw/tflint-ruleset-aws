package aws

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	awsbase "github.com/hashicorp/aws-sdk-go-base/v2"
	"github.com/mitchellh/go-homedir"
)

func Test_getBaseConfig(t *testing.T) {
	home, err := homedir.Expand("~/")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		Name     string
		Creds    Credentials
		Expected *awsbase.Config
	}{
		{
			Name: "static credentials",
			Creds: Credentials{
				AccessKey: "AWS_ACCESS_KEY",
				SecretKey: "AWS_SECRET_KEY",
				Region:    "us-east-1",
			},
			Expected: &awsbase.Config{
				AccessKey:              "AWS_ACCESS_KEY",
				SecretKey:              "AWS_SECRET_KEY",
				Region:                 "us-east-1",
				CallerDocumentationURL: "https://github.com/terraform-linters/tflint-ruleset-aws/blob/master/docs/deep_checking.md",
				CallerName:             "tflint-ruleset-aws",
			},
		},
		{
			Name: "shared credentials",
			Creds: Credentials{
				Profile:   "default",
				CredsFile: "~/.aws/creds",
				Region:    "us-east-1",
			},
			Expected: &awsbase.Config{
				Profile:                "default",
				SharedCredentialsFiles: []string{filepath.Join(home, ".aws", "creds")},
				Region:                 "us-east-1",
				CallerDocumentationURL: "https://github.com/terraform-linters/tflint-ruleset-aws/blob/master/docs/deep_checking.md",
				CallerName:             "tflint-ruleset-aws",
			},
		},
		{
			Name: "assume role chain",
			Creds: Credentials{
				AssumeRoles: []AssumeRole{
					{
						RoleARN:     "arn:aws:iam::111111111111:role/INITIAL_ROLE",
						SessionName: "INITIAL_SESSION",
					},
					{
						RoleARN:    "arn:aws:iam::222222222222:role/FINAL_ROLE",
						ExternalID: "EXTERNAL_ID",
						Policy:     "POLICY",
					},
				},
			},
			Expected: &awsbase.Config{
				AssumeRole: []awsbase.AssumeRole{
					{
						RoleARN:     "arn:aws:iam::111111111111:role/INITIAL_ROLE",
						SessionName: "INITIAL_SESSION",
					},
					{
						RoleARN:    "arn:aws:iam::222222222222:role/FINAL_ROLE",
						ExternalID: "EXTERNAL_ID",
						Policy:     "POLICY",
					},
				},
				CallerDocumentationURL: "https://github.com/terraform-linters/tflint-ruleset-aws/blob/master/docs/deep_checking.md",
				CallerName:             "tflint-ruleset-aws",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			base, err := getBaseConfig(tc.Creds)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(tc.Expected, base) {
				t.Fatalf("diff=%s", cmp.Diff(tc.Expected, base))
			}
		})
	}
}

func Test_Merge(t *testing.T) {
	cases := []struct {
		Name     string
		Self     Credentials
		Other    Credentials
		Expected Credentials
	}{
		{
			Name: "self is empty",
			Self: Credentials{},
			Other: Credentials{
				AccessKey: "AWS_ACCESS_KEY",
				SecretKey: "AWS_SECRET_KEY",
				Profile:   "default",
				CredsFile: "~/.aws/creds",
				AssumeRoles: []AssumeRole{
					{
						RoleARN:     "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME",
						SessionName: "SESSION_NAME",
						ExternalID:  "EXTERNAL_ID",
						Policy:      "POLICY_NAME",
					},
				},
				Region: "us-east-1",
			},
			Expected: Credentials{
				AccessKey: "AWS_ACCESS_KEY",
				SecretKey: "AWS_SECRET_KEY",
				Profile:   "default",
				CredsFile: "~/.aws/creds",
				AssumeRoles: []AssumeRole{
					{
						RoleARN:     "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME",
						SessionName: "SESSION_NAME",
						ExternalID:  "EXTERNAL_ID",
						Policy:      "POLICY_NAME",
					},
				},
				Region: "us-east-1",
			},
		},
		{
			Name: "other is empty",
			Self: Credentials{
				AccessKey: "AWS_ACCESS_KEY_2",
				SecretKey: "AWS_SECRET_KEY_2",
				Profile:   "staging",
				CredsFile: "~/.aws/creds_stg",
				AssumeRoles: []AssumeRole{
					{
						RoleARN:     "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME",
						SessionName: "SESSION_NAME",
						ExternalID:  "EXTERNAL_ID",
						Policy:      "POLICY_NAME",
					},
				},
				Region: "ap-northeast-1",
			},
			Other: Credentials{},
			Expected: Credentials{
				AccessKey: "AWS_ACCESS_KEY_2",
				SecretKey: "AWS_SECRET_KEY_2",
				Profile:   "staging",
				CredsFile: "~/.aws/creds_stg",
				AssumeRoles: []AssumeRole{
					{
						RoleARN:     "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME",
						SessionName: "SESSION_NAME",
						ExternalID:  "EXTERNAL_ID",
						Policy:      "POLICY_NAME",
					},
				},
				Region: "ap-northeast-1",
			},
		},
		{
			Name: "merged",
			Self: Credentials{
				AccessKey: "AWS_ACCESS_KEY_2",
				SecretKey: "AWS_SECRET_KEY_2",
				Profile:   "staging",
				CredsFile: "~/.aws/creds_stg",
				AssumeRoles: []AssumeRole{
					{
						RoleARN:     "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME_2",
						SessionName: "SESSION_NAME_2",
						ExternalID:  "EXTERNAL_ID_2",
						Policy:      "POLICY_NAME_2",
					},
				},
				Region: "ap-northeast-1",
			},
			Other: Credentials{
				AccessKey: "AWS_ACCESS_KEY",
				SecretKey: "AWS_SECRET_KEY",
				Profile:   "default",
				CredsFile: "~/.aws/creds",
				AssumeRoles: []AssumeRole{
					{
						RoleARN:     "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME",
						SessionName: "SESSION_NAME",
						ExternalID:  "EXTERNAL_ID",
						Policy:      "POLICY_NAME",
					},
				},
				Region: "us-east-1",
			},
			Expected: Credentials{
				AccessKey: "AWS_ACCESS_KEY",
				SecretKey: "AWS_SECRET_KEY",
				Profile:   "default",
				CredsFile: "~/.aws/creds",
				AssumeRoles: []AssumeRole{
					{
						RoleARN:     "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME",
						SessionName: "SESSION_NAME",
						ExternalID:  "EXTERNAL_ID",
						Policy:      "POLICY_NAME",
					},
				},
				Region: "us-east-1",
			},
		},
		{
			Name: "single override is merged into final role",
			Self: Credentials{
				AssumeRoles: []AssumeRole{
					{RoleARN: "arn:aws:iam::111111111111:role/INITIAL_ROLE", SessionName: "INITIAL_SESSION"},
					{RoleARN: "arn:aws:iam::222222222222:role/FINAL_ROLE", SessionName: "OLD_SESSION", ExternalID: "OLD_EXTERNAL_ID"},
				},
			},
			Other: Credentials{
				AssumeRoles: []AssumeRole{{SessionName: "NEW_SESSION", ExternalID: "NEW_EXTERNAL_ID"}},
			},
			Expected: Credentials{
				AssumeRoles: []AssumeRole{
					{RoleARN: "arn:aws:iam::111111111111:role/INITIAL_ROLE", SessionName: "INITIAL_SESSION"},
					{RoleARN: "arn:aws:iam::222222222222:role/FINAL_ROLE", SessionName: "NEW_SESSION", ExternalID: "NEW_EXTERNAL_ID"},
				},
			},
		},
		{
			Name: "override chain replaces provider chain",
			Self: Credentials{
				AssumeRoles: []AssumeRole{
					{
						RoleARN: "arn:aws:iam::111111111111:role/PROVIDER_ROLE",
					},
				},
			},
			Other: Credentials{
				AssumeRoles: []AssumeRole{
					{
						RoleARN: "arn:aws:iam::222222222222:role/INITIAL_OVERRIDE_ROLE",
					},
					{
						RoleARN: "arn:aws:iam::333333333333:role/FINAL_OVERRIDE_ROLE",
					},
				},
			},
			Expected: Credentials{
				AssumeRoles: []AssumeRole{
					{
						RoleARN: "arn:aws:iam::222222222222:role/INITIAL_OVERRIDE_ROLE",
					},
					{
						RoleARN: "arn:aws:iam::333333333333:role/FINAL_OVERRIDE_ROLE",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		ret := tc.Self.Merge(tc.Other)
		if !cmp.Equal(tc.Expected, ret) {
			t.Fatalf("Failed `%s` test: Diff=%s", tc.Name, cmp.Diff(tc.Expected, ret))
		}
	}
}
