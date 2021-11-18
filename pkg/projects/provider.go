package projects

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"net/http"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var Version = "0.0.1"

// Provider Projects provider that supports configuration via username+password or a token
// Supported resources are repos, users, groups, replications, and permissions
func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"PROJECTS_URL", "JFROG_URL"}, "http://localhost:8081"),
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"access_token": {
				Type:             schema.TypeString,
				Optional:         true,
				Sensitive:        true,
				DefaultFunc:      schema.MultiEnvDefaultFunc([]string{"PROJECTS_ACCESS_TOKEN", "JFROG_ACCESS_TOKEN"}, ""),
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      "This is a Bearer token that can be given to you by your admin under `Identity and Access`",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"project": projectResource(),
		},
	}

	p.ConfigureContextFunc = func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			terraformVersion = "0.13+compatible"
		}
		configuration, err := providerConfigure(data, terraformVersion)
		return configuration, diag.FromErr(err)
	}

	return p
}

type ProjectClient struct {
	Get func(id string) (Project, error)
}

func buildResty(URL string) (*resty.Client, error) {

	u, err := url.ParseRequestURI(URL)
	if err != nil {
		return nil, err
	}

	baseUrl := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	restyBase := resty.New().SetHostURL(baseUrl).OnAfterResponse(func(client *resty.Client, response *resty.Response) error {
		if response == nil {
			return fmt.Errorf("no response found")
		}

		if response.StatusCode() >= http.StatusBadRequest {
			return fmt.Errorf("\n%d %s %s\n%s", response.StatusCode(), response.Request.Method, response.Request.URL, string(response.Body()[:]))
		}

		return nil
	}).
		SetHeader("content-type", "application/json").
		SetHeader("accept", "*/*").
		SetHeader("user-agent", "jfrog/terraform-provider-projects:"+Version).
		SetRetryCount(5)

	restyBase.DisableWarn = true

	return restyBase, nil
}

func mkProjectClient(resty *resty.Client) (ProjectClient, error) {
	// groupsChan := make(chan Group)
	// defer close(groupsChan)
	//
	// var getGroup = func(prjId, grpId string) {
	// 	var group Group
	// 	resty.R().SetResult(&group).SetPathParam("group", grpId).
	// 		SetPathParam("project", prjId).
	// 		Get("/access/api/v1/projects/{project}/groups/{group}")
	// 	//if err != nil {
	// 	//	return nil, err // make an error channel or a new type?
	// 	//}
	// 	groupsChan <- group
	// }
	//
	// var getGroups = func(prjId string) ([]Group, error) {
	// 	var result []Group
	//
	// 	_, err := resty.R().SetResult(&result).Get(fmt.Sprintf("/access/api/v1/projects/%s/groups/", prjId))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	//
	// 	for group := range groupsChan {
	// 		go getGroup(prjId, group.Id())
	// 	}
	//
	// 	return result, nil
	// }
	//
	// roleChan := make(chan Role)
	// defer close(roleChan)
	// var getRole = func(prjId, roleId string) {
	// 	var role Role
	// 	resty.R().SetResult(&role).SetPathParam("role", roleId).
	// 		SetPathParam("project", prjId).
	// 		Get("/access/api/v1/projects/{project}/groups/{role}")
	// 	//if err != nil {
	// 	//	return nil, err // make an error channel or a new type?
	// 	//}
	// 	roleChan <- role
	// }
	//
	// var getRoles = func(prjId string) ([]Role, error) {
	// 	var result []Role
	//
	// 	_, err := resty.R().SetResult(result).SetPathParam("prjKey", prjId).Get("/access/api/v1/projects/{prjKey}/roles/")
	// 	if err != nil {
	// 		return nil, err
	// 	}
	//
	// 	for group := range groupsChan {
	// 		go getRole(prjId, group.Id())
	// 	}
	//
	// 	return result, nil
	// }

	client := ProjectClient{
		Get: func(id string) (Project, error) {
			project := Project{}
			// groups, err := getGroups(id)
			// roles, err := getRoles(id)
			// if err != nil {
			// 	return Project{}, err
			// }

			_, err := resty.R().SetResult(&project).Get("/access/api/v1/projects/" + id)
			if err != nil {
				return Project{}, err
			}

			// result.Groups = groups
			// result.Roles = roles

			return project, nil
		},
	}
	// or, remove the struct and just return the function
	return client, nil
}

func addAuthToResty(client *resty.Client, accessToken string) (*resty.Client, error) {
	if accessToken != "" {
		return client.SetAuthToken(accessToken), nil
	}
	return nil, fmt.Errorf("no authentication details supplied")
}

func providerConfigure(d *schema.ResourceData, terraformVersion string) (interface{}, error) {
	URL, ok := d.GetOk("url")
	if URL == nil || URL == "" || !ok {
		return nil, fmt.Errorf("you must supply a URL")
	}

	restyBase, err := buildResty(URL.(string))
	if err != nil {
		return nil, err
	}
	accessToken := d.Get("access_token").(string)

	restyBase, err = addAuthToResty(restyBase, accessToken)
	if err != nil {
		return nil, err
	}

	// _, err := mkProjectClient(restyBase)
	// if err != nil {
	// 	return nil, err
	// }

	_, err = sendUsageRepo(restyBase, terraformVersion)
	if err != nil {
		return nil, err
	}

	return restyBase, nil

}

func sendUsageRepo(restyBase *resty.Client, terraformVersion string) (interface{}, error) {
	type Feature struct {
		FeatureId string `json:"featureId"`
	}
	type UsageStruct struct {
		ProductId string    `json:"productId"`
		Features  []Feature `json:"features"`
	}
	_, err := restyBase.R().SetBody(UsageStruct{
		"terraform-provider-projects/" + Version,
		[]Feature{
			{FeatureId: "Partner/ACC-007450"},
			{FeatureId: "Terraform/" + terraformVersion},
		},
	}).Post("artifactory/api/system/usage")

	if err != nil {
		return nil, fmt.Errorf("unable to report usage %s", err)
	}
	return nil, nil
}
