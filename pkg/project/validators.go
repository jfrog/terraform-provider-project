package project

import (
	"fmt"
)

func maxLength(length int) func(i interface{}, k string) ([]string, []error) {
	return func(value interface{}, k string) ([]string, []error) {
		if len(value.(string)) > length {
			return nil, []error{fmt.Errorf("string must be less than or equal %d characters long", length)}
		}
		return nil, nil
	}
}
