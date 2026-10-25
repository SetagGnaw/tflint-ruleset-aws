package aws

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
)

func TestConfigAssumeRoleChain(t *testing.T) {
	file, diags := hclsyntax.ParseConfig([]byte(`
assume_role {
  role_arn     = "arn:aws:iam::111111111111:role/INITIAL_ROLE"
  session_name = "INITIAL_SESSION"
}

assume_role {
  role_arn    = "arn:aws:iam::222222222222:role/FINAL_ROLE"
  external_id = "EXTERNAL_ID"
  policy      = "POLICY"
}
`), ".tflint.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	config := &Config{}
	body, diags := hclext.Content(file.Body, hclext.ImpliedBodySchema(config))
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	if diags := hclext.DecodeBody(body, nil, config); diags.HasErrors() {
		t.Fatal(diags)
	}

	want := []AssumeRole{
		{
			RoleARN:     "arn:aws:iam::111111111111:role/INITIAL_ROLE",
			SessionName: "INITIAL_SESSION",
		},
		{
			RoleARN:    "arn:aws:iam::222222222222:role/FINAL_ROLE",
			ExternalID: "EXTERNAL_ID",
			Policy:     "POLICY",
		},
	}
	if diff := cmp.Diff(want, config.toCredentials().AssumeRoles); diff != "" {
		t.Fatalf("unexpected assume role chain (-want +got):\n%s", diff)
	}
}
