package gemini

import (
	"encoding/json"
	"net/url"

	"github.com/google/uuid"
)

// BuildStreamGeneratePayload constructs the 80-element nested array required by Google StreamGenerate.
func BuildStreamGeneratePayload(prompt string, modelID int, thinkMode int, temporaryChats bool, fileRefs []string) (string, error) {
	inner := make([]interface{}, 80)

	var promptElement interface{}
	if len(fileRefs) > 0 {
		refs := make([]interface{}, len(fileRefs))
		for i, ref := range fileRefs {
			refs[i] = []interface{}{nil, nil, ref}
		}
		promptElement = []interface{}{prompt, 0, nil, refs, nil, nil, 0}
	} else {
		promptElement = []interface{}{prompt, 0, nil, nil, nil, nil, 0}
	}

	inner[0] = promptElement
	inner[1] = []string{"en"}
	inner[2] = []interface{}{"", "", "", nil, nil, nil, nil, nil, nil, ""}
	inner[6] = []int{0}
	inner[7] = 1
	inner[10] = 1
	inner[11] = 0
	inner[17] = []interface{}{[]int{thinkMode}}
	inner[18] = 0
	inner[27] = 1
	inner[30] = []int{4}

	if temporaryChats {
		inner[41] = []int{1}
		inner[45] = 1
	} else {
		inner[41] = []int{2}
	}

	inner[53] = 0
	inner[59] = uuid.New().String()
	inner[61] = []interface{}{}
	inner[68] = 1
	inner[79] = modelID

	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return "", err
	}

	outer := []interface{}{nil, string(innerJSON)}
	outerJSON, err := json.Marshal(outer)
	if err != nil {
		return "", err
	}

	return string(outerJSON), nil
}

// EncodeFormBody serializes the payload parameter `f.req` and optional XSRF token `at`.
func EncodeFormBody(outerJSON string, xsrfToken string) string {
	form := url.Values{}
	form.Set("f.req", outerJSON)
	if xsrfToken != "" {
		form.Set("at", xsrfToken)
	}
	return form.Encode()
}
