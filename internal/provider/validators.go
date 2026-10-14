package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type oneOfValidator struct {
	values []string
}

func stringOneOf(values ...string) validator.String {
	return oneOfValidator{values: values}
}

func (v oneOfValidator) Description(_ context.Context) string {
	return fmt.Sprintf("une des valeurs %v", v.values)
}

func (v oneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v oneOfValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	for _, allowed := range v.values {
		if val == allowed {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Valeur invalide",
		fmt.Sprintf("%q n'est pas dans %v", val, v.values),
	)
}
