package projects

import (
	"github.com/gorhill/cronexpr"
)


func validateCron(value interface{}, key string) (ws []string, es []error) {
	_, err := cronexpr.Parse(value.(string))
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}


