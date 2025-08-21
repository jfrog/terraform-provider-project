package project

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/samber/lo"
)

type Equatable interface {
	util.Identifiable
	Equals(other Equatable) bool
}

func RetryOnSpecificMsgBody(matchString string) func(response *resty.Response, err error) bool {
	return func(response *resty.Response, err error) bool {
		return regexp.MustCompile(matchString).MatchString(string(response.Body()[:]))
	}
}

type ProjectError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ProjectError) String() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

type ProjectErrorsResponse struct {
	Errors []ProjectError `json:"errors"`
}

func (r ProjectErrorsResponse) String() string {
	errs := lo.Reduce(r.Errors, func(err string, item ProjectError, _ int) string {
		if err == "" {
			return item.String()
		} else {
			return fmt.Sprintf("%s, %s", err, item.String())
		}
	}, "")
	return errs
}

const ProjectRepositoryStatusEndpoint = "access/api/v1/projects/_/repositories/{repo_key}"

type ProjectRepositoryStatusAPIModel struct {
	ResourceName          string   `json:"resource_name"`
	Environments          []string `json:"environments"`
	SharedWithProjects    []string `json:"shared_with_projects"`
	SharedWithAllProjects bool     `json:"shared_with_all_projects"`
	SharedReadOnly        bool     `json:"shared_read_only"`
	AssignedTo            string   `json:"assigned_to"`
}

var GlobalMutex = newMutexKV()

// mutexKV is a simple key/value store for arbitrary mutexes. It can be used to
// serialize changes across arbitrary collaborators that share knowledge of the
// keys they must serialize on.
type mutexKV struct {
	lock  sync.Mutex
	store map[string]*sync.Mutex
}

// Locks the mutex for the given key. Caller is responsible for calling Unlock
// for the same key
func (m *mutexKV) Lock(key string) {
	m.get(key).Lock()
}

// Unlock the mutex for the given key. Caller must have called Lock for the same key first
func (m *mutexKV) Unlock(key string) {
	m.get(key).Unlock()
}

// Returns a mutex for the given key, no guarantee of its lock status
func (m *mutexKV) get(key string) *sync.Mutex {
	m.lock.Lock()
	defer m.lock.Unlock()
	mutex, ok := m.store[key]
	if !ok {
		mutex = &sync.Mutex{}
		m.store[key] = mutex
	}
	return mutex
}

// Returns a properly initialized MutexKV
func newMutexKV() *mutexKV {
	return &mutexKV{
		store: make(map[string]*sync.Mutex),
	}
}
