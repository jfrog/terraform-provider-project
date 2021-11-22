package projects

import (
	"fmt"

	"github.com/gorhill/cronexpr"
)

func validateCron(value interface{}, key string) (ws []string, es []error) {
	_, err := cronexpr.Parse(value.(string))
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func maxLength(length int) func(i interface{}, k string) ([]string, []error) {
	return func(value interface{}, k string) ([]string, []error) {
		if len(value.(string)) > length {
			return nil, []error{fmt.Errorf("string must be less than or equal %d characters long", length)}
		}
		return nil, nil
	}
}
