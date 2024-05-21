package project

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func maxLength(length int) func(i interface{}, k string) ([]string, []error) {
	return func(value interface{}, k string) ([]string, []error) {
		if len(value.(string)) > length {
			return nil, []error{fmt.Errorf("string must be less than or equal %d characters long", length)}
		}
		return nil, nil
	}
}

func int64Between(min, max int64) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (warnings []string, errors []error) {
		v1, ok := i.(int)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %s to be integer", k))
			return warnings, errors
		}

		v2 := int64(v1)
		if v2 < min || v2 > max {
			errors = append(errors, fmt.Errorf("expected %s to be in the range (%d - %d), got %d", k, min, max, v2))
			return warnings, errors
		}

		return warnings, errors
	}
}
