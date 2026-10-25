package aws

import (
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// AwsProviderBlockSchema is a schema of `aws` provider block
var AwsProviderBlockSchema = &hclext.BodySchema{
	Attributes: []hclext.AttributeSchema{
		{Name: "access_key"},
		{Name: "secret_key"},
		{Name: "profile"},
		{Name: "shared_credentials_file"},
		{Name: "region"},
		{Name: "alias"},
	},
	Blocks: []hclext.BlockSchema{
		{
			Type: "assume_role",
			Body: AwsProviderAssumeRoleBlockShema,
		},
	},
}

// AwsProviderAssumeRoleBlockShema is a schema of `assume_role` block
var AwsProviderAssumeRoleBlockShema = &hclext.BodySchema{
	Attributes: []hclext.AttributeSchema{
		{Name: "role_arn", Required: true},
		{Name: "session_name"},
		{Name: "external_id"},
		{Name: "policy"},
	},
}

// GetCredentialsFromProvider retrieves credentials from the "provider" block in the Terraform configuration
func GetCredentialsFromProvider(runner tflint.Runner) (map[string]Credentials, error) {
	providers, err := runner.GetModuleContent(
		&hclext.BodySchema{
			Blocks: []hclext.BlockSchema{
				{
					Type:       "provider",
					LabelNames: []string{"name"},
					Body:       AwsProviderBlockSchema,
				},
			},
		},
		&tflint.GetModuleContentOption{ModuleCtx: tflint.RootModuleCtxType},
	)
	if err != nil {
		return nil, err
	}

	m := map[string]Credentials{}
	opts := &tflint.EvaluateExprOption{ModuleCtx: tflint.RootModuleCtxType}

	type stringAttribute struct {
		name        string
		destination *string
	}
	evaluateStringAttributes := func(attributes hclext.Attributes, destinations []stringAttribute) error {
		for _, destination := range destinations {
			attr, exists := attributes[destination.name]
			if !exists {
				continue
			}

			if err := runner.EvaluateExpr(attr.Expr, func(value string) error {
				*destination.destination = value
				return nil
			}, opts); err != nil {
				return err
			}
		}
		return nil
	}

	for _, provider := range providers.Blocks {
		if provider.Labels[0] != "aws" {
			continue
		}

		creds := Credentials{}

		if err := evaluateStringAttributes(provider.Body.Attributes, []stringAttribute{
			{name: "access_key", destination: &creds.AccessKey},
			{name: "secret_key", destination: &creds.SecretKey},
			{name: "profile", destination: &creds.Profile},
			{name: "shared_credentials_file", destination: &creds.CredsFile},
			{name: "region", destination: &creds.Region},
		}); err != nil {
			return nil, err
		}

		for _, assumeRole := range provider.Body.Blocks {
			assumeRoleConfig := AssumeRole{}
			if err := evaluateStringAttributes(assumeRole.Body.Attributes, []stringAttribute{
				{name: "role_arn", destination: &assumeRoleConfig.RoleARN},
				{name: "session_name", destination: &assumeRoleConfig.SessionName},
				{name: "external_id", destination: &assumeRoleConfig.ExternalID},
				{name: "policy", destination: &assumeRoleConfig.Policy},
			}); err != nil {
				return nil, err
			}
			creds.AssumeRoles = append(creds.AssumeRoles, assumeRoleConfig)
		}
		if attr, exists := provider.Body.Attributes["alias"]; exists {
			if err := runner.EvaluateExpr(attr.Expr, func(alias string) error {
				m[alias] = creds
				return nil
			}, opts); err != nil {
				return nil, err
			}
		} else {
			m["aws"] = creds
		}
	}

	return m, nil
}
