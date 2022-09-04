package project

import (
	"math"
	"net/http"
	"regexp"

	"github.com/go-resty/resty/v2"
	"github.com/jfrog/terraform-provider-shared/util"
)

func BytesToGibibytes(bytes int) int {
	if bytes <= -1 {
		return -1
	}

	return int(bytes / int(math.Pow(1024, 3)))
}

func GibibytesToBytes(bytes int) int {
	if bytes <= -1 {
		return -1
	}

	return bytes * int(math.Pow(1024, 3))
}

type Equatable interface {
	util.Identifiable
	Equals(other Equatable) bool
}

func retryOnSpecificMsgBody(matchString string) func(response *resty.Response, err error) bool {
	return func(response *resty.Response, err error) bool {
		var responseBodyRegex = regexp.MustCompile(matchString)
		return responseBodyRegex.MatchString(string(response.Body()[:]))
	}
}

var retryOnServiceUnavailable = func(response *resty.Response, err error) bool {
	return response.StatusCode() == http.StatusServiceUnavailable
}
