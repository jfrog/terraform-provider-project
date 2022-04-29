package project

import (
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/go-resty/resty/v2"
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

type Identifiable interface {
	Id() string
}

type Equatable interface {
	Identifiable
	Equals(other Equatable) bool
}

var getBoolEnvVar = func(key string, fallback bool) bool {
	value, exists := os.LookupEnv(key)
	if exists {
		boolValue, err := strconv.ParseBool(value)
		if err == nil {
			return boolValue
		}
	}
	return fallback
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
