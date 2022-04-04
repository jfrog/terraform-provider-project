package project

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testAccProviders() map[string]func() (*schema.Provider, error) {
	return map[string]func() (*schema.Provider, error){
		"project": func() (*schema.Provider, error) {
			return Provider(), nil
		},
	}
}

func fmtMapToHcl(fields map[string]interface{}) string {
	var allPairs []string
	max := float64(0)
	for key := range fields {
		max = math.Max(max, float64(len(key)))
	}
	for key, value := range fields {
		hcl := toHclFormat(value)
		format := toHclFormatString(3, int(max), value)
		allPairs = append(allPairs, fmt.Sprintf(format, key, hcl))
	}

	return strings.Join(allPairs, "\n")
}

func toHclFormatString(tabs, max int, value interface{}) string {
	prefix := ""
	suffix := ""
	delimeter := "="
	if reflect.TypeOf(value).Kind() == reflect.Map {
		delimeter = ""
		prefix = "{"
		suffix = "}"
	}
	return fmt.Sprintf("%s%%-%ds %s %s%s%s", strings.Repeat("\t", tabs), max, delimeter, prefix, "%s", suffix)
}

func mapToTestChecks(fqrn string, fields map[string]interface{}) []resource.TestCheckFunc {
	var result []resource.TestCheckFunc
	for key, value := range fields {
		switch reflect.TypeOf(value).Kind() {
		case reflect.Slice:
			for i, lv := range value.([]interface{}) {
				result = append(result, resource.TestCheckResourceAttr(
					fqrn,
					fmt.Sprintf("%s.%d", key, i),
					fmt.Sprintf("%v", lv),
				))
			}
		case reflect.Map:
			// this also gets generated, but it's value is '1', which is also the size. So, I don't know
			// what it means
			// content_synchronisation.0.%
			resource.TestCheckResourceAttr(
				fqrn,
				fmt.Sprintf("%s.#", key),
				fmt.Sprintf("%d", len(value.(map[string]interface{}))),
			)
		default:
			result = append(result, resource.TestCheckResourceAttr(fqrn, key, fmt.Sprintf(`%v`, value)))
		}
	}
	return result
}

func toHclFormat(thing interface{}) string {
	switch thing.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, thing.(string))
	case []interface{}:
		var result []string
		for _, e := range thing.([]interface{}) {
			result = append(result, toHclFormat(e))
		}
		return fmt.Sprintf("[%s]", strings.Join(result, ","))
	case map[string]interface{}:
		return fmt.Sprintf("\n\t%s\n\t\t\t\t", fmtMapToHcl(thing.(map[string]interface{})))
	default:
		return fmt.Sprintf("%v", thing)
	}
}

type CheckFun func(id string, request *resty.Request) (*resty.Response, error)

func verifyDeleted(id string, check CheckFun) func(*terraform.State) error {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[id]

		if !ok {
			return fmt.Errorf("error: Resource id [%s] not found", id)
		}
		provider, _ := testAccProviders()["project"]()
		provider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
		client := provider.Meta().(*resty.Client)
		resp, err := check(rs.Primary.ID, client.R())
		if err != nil {
			if resp != nil {
				switch resp.StatusCode() {
				case http.StatusNotFound, http.StatusBadRequest:
					return nil
				}
			}
			return err
		}
		return fmt.Errorf("error: %s still exists", rs.Primary.ID)
	}
}

func mkNames(name, resource string) (int, string, string) {
	id := randomInt()
	n := fmt.Sprintf("%s%d", name, id)
	return id, fmt.Sprintf("%s.%s", resource, n), n
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

var randomInt = func() func() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Int
}()

func randBool() bool {
	return randomInt()%2 == 0
}

func randSelect(items ...interface{}) interface{} {
	return items[randomInt()%len(items)]
}

func createTestUser(t *testing.T, name string, email string) {

	type ArtifactoryUser struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Admin    bool   `json:"admin"`
	}

	restyClient := getTestResty(t)

	user := ArtifactoryUser{
		Email:    email,
		Password: "Password1",
		Admin:    false,
	}

	_, err := restyClient.R().SetBody(user).Put("/artifactory/api/security/users/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

func deleteTestUser(t *testing.T, name string) {
	restyClient := getTestResty(t)

	_, err := restyClient.R().Delete("/artifactory/api/security/users/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

func createTestGroup(t *testing.T, name string) {

	type ArtifactoryGroup struct {
		Name string `json:"name"`
	}

	restyClient := getTestResty(t)

	group := ArtifactoryGroup{
		Name: name,
	}

	_, err := restyClient.R().SetBody(group).Put("/artifactory/api/security/groups/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

func deleteTestGroup(t *testing.T, name string) {
	restyClient := getTestResty(t)

	_, err := restyClient.R().Delete("/artifactory/api/security/groups/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

var alwaysRetry = func(response *resty.Response, err error) bool {
	return true
}

func createTestRepo(t *testing.T, name string) {

	type ArtifactoryRepo struct {
		Name   string `json:"key"`
		RClass string `json:"rclass"`
	}

	restyClient := getTestResty(t)

	repo := ArtifactoryRepo{
		Name:   name,
		RClass: "local",
	}

	_, err := cloneResty(restyClient).
		SetRetryCount(500).
		SetRetryWaitTime(5 * time.Second).
		SetRetryMaxWaitTime(20 * time.Second).
		R().
		SetBody(repo).
		Put("/artifactory/api/repositories/" + name)

	if err != nil {
		//t.Fatal(err)
		fmt.Println(err)
	}
}

func deleteTestRepo(t *testing.T, name string) {
	restyClient := getTestResty(t)

	_, err := cloneResty(restyClient).
		SetRetryCount(500).
		SetRetryWaitTime(5 * time.Second).
		SetRetryMaxWaitTime(20 * time.Second).
		R().
		Delete("/artifactory/api/repositories/" + name)

	if err != nil {
		//t.Fatal(err)
		fmt.Println(err)
	}
}
