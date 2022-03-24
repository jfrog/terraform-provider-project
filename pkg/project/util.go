package project

import (
	"bytes"
	"log"
	"math"
	"text/template"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ResourceData struct{ *schema.ResourceData }

func (d *ResourceData) getString(key string) string {
	if v, ok := d.GetOk(key); ok {
		return v.(string)
	}
	return ""
}

func (d *ResourceData) getBool(key string) bool {
	if v, ok := d.GetOkExists(key); ok {
		return v.(bool)
	}
	return false
}

func (d *ResourceData) getInt(key string) int {
	if v, ok := d.GetOkExists(key); ok {
		return v.(int)
	}
	return 0
}

func (d *ResourceData) getSet(key string) []string {
	if v, ok := d.GetOkExists(key); ok {
		arr := castToStringArr(v.(*schema.Set).List())
		return arr
	}
	return nil
}
func (d *ResourceData) getList(key string) []string {
	if v, ok := d.GetOkExists(key); ok {
		arr := castToStringArr(v.([]interface{}))
		return arr
	}
	return []string{}
}
func (d *ResourceData) getListRef(key string) *[]string {
	if v, ok := d.GetOkExists(key); ok {
		arr := castToStringArr(v.([]interface{}))
		return &arr
	}
	return new([]string)
}

func castToStringArr(arr []interface{}) []string {
	cpy := make([]string, 0, len(arr))
	for _, r := range arr {
		cpy = append(cpy, r.(string))
	}

	return cpy
}

func castToInterfaceArr(arr []string) []interface{} {
	cpy := make([]interface{}, 0, len(arr))
	for _, r := range arr {
		cpy = append(cpy, r)
	}

	return cpy
}

func mergeMaps(schemata ...map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for _, schma := range schemata {
		for k, v := range schma {
			result[k] = v
		}
	}
	return result
}

func mergeSchema(schemata ...map[string]*schema.Schema) map[string]*schema.Schema {
	result := map[string]*schema.Schema{}
	for _, schma := range schemata {
		for k, v := range schma {
			result[k] = v
		}
	}
	return result
}

func executeTemplate(name, temp string, fields interface{}) string {
	var tpl bytes.Buffer
	if err := template.Must(template.New(name).Parse(temp)).Execute(&tpl, fields); err != nil {
		panic(err)
	}

	return tpl.String()
}

type Lens func(key string, value interface{}) []error

func mkLens(d *schema.ResourceData) Lens {
	var errors []error
	return func(key string, value interface{}) []error {
		if err := d.Set(key, value); err != nil {
			errors = append(errors, err)
		}
		return errors
	}
}

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
	Equals(other Identifiable) bool
}

func contains(as []Equatable, b Equatable) bool {
	log.Printf("[DEBUG] contains")
	log.Printf("[TRACE] as: %+v\n", as)
	log.Printf("[TRACE] b: %+v\n", b)

	for _, a := range as {
		log.Printf("[TRACE] a: %+v\n", a)
		log.Printf("[TRACE] a.Equals(b): %+v\n", a.Equals(b))
		if a.Equals(b) {
			return true
		}
	}
	return false
}

var apply = func(predicate func(bs []Equatable, a Equatable) bool) func(as []Equatable, bs []Equatable) []Equatable {
	return func(as []Equatable, bs []Equatable) []Equatable {
		var results []Equatable

		// Not the most efficient way to determine the slices intersection but this suffices for the small-ish number of items
		for _, a := range as {
			if predicate(bs, a) {
				results = append(results, a)
			}
		}

		return results
	}
}

var intersection = apply(func(bs []Equatable, a Equatable) bool {
	return contains(bs, a)
})

var difference = apply(func(bs []Equatable, a Equatable) bool {
	return !contains(bs, a)
})
