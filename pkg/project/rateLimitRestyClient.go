package project

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"golang.org/x/time/rate"
	"time"
)

var rateLimiterMap = map[string]*rate.Limiter{
	"GLOBAL":                     rate.NewLimiter(rate.Every(1*time.Second), 1),
	"REPO_CREATE_API":            rate.NewLimiter(rate.Every(10*time.Second), 10),
	"REPO_DELETE_API":            rate.NewLimiter(rate.Every(10*time.Second), 10),
	"ATTACH_REPO_TO_PROJECT_API": rate.NewLimiter(rate.Every(1*time.Second), 1),
	"DETACH_REPO_TO_PROJECT_API": rate.NewLimiter(rate.Every(1*time.Second), 1),
}

type ExRequest struct {
	r *resty.Request
}

func (req *ExRequest) Limit(rateLimiterKey string) *resty.Request {
	rateLimiter := rateLimiterMap[rateLimiterKey]
	ctx := context.Background()
	err := rateLimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		fmt.Println(err)
	}
	return req.r
}
