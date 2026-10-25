package aws

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func TestGetCredentialsFromProvider_AssumeRoleChain(t *testing.T) {
	runner := helper.TestRunner(t, map[string]string{
		"provider.tf": `
provider "aws" {
  region = "us-east-1"

  assume_role {
    role_arn     = "arn:aws:iam::111111111111:role/INITIAL_ROLE"
    session_name = "INITIAL_SESSION"
  }

  assume_role {
    role_arn    = "arn:aws:iam::222222222222:role/FINAL_ROLE"
    external_id = "EXTERNAL_ID"
    policy      = "POLICY"
  }
}
`,
	})

	got, err := GetCredentialsFromProvider(runner)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]Credentials{
		"aws": {
			Region: "us-east-1",
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
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected credentials (-want +got):\n%s", diff)
	}
}
