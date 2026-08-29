package gemini

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	bardErrRegex = regexp.MustCompile(`BardErrorInfo\s*\[(\d+)\]`)
	codeArtifactRegex = regexp.MustCompile("(?s)```(?:python|javascript|text)\\?code_(?:reference|stdout)&code_event_index=\\d+\\n.*?```\\n?")
)

// CleanGeminiText strips internal execution artifacts from the Gemini response stream.
func CleanGeminiText(text string, stripSpace bool) string {
	cleaned := codeArtifactRegex.ReplaceAllString(text, "")
	if stripSpace {
		return strings.TrimSpace(cleaned)
	}
	return cleaned
}

// ParseChunkLine extracts candidate text from a raw response line.
func ParseChunkLine(line string, prevText string) (delta string, newPrevText string, err error) {
	if match := bardErrRegex.FindStringSubmatch(line); len(match) > 1 {
		return "", prevText, fmt.Errorf("gemini upstream rejected request: BardErrorInfo [%s]", match[1])
	}

	if !strings.Contains(line, `"wrb.fr"`) {
		return "", prevText, nil
	}

	var arr []interface{}
	if jsonErr := json.Unmarshal([]byte(line), &arr); jsonErr != nil || len(arr) == 0 {
		return "", prevText, nil
	}

	firstElem, ok := arr[0].([]interface{})
	if !ok || len(firstElem) < 3 {
		return "", prevText, nil
	}

	innerStr, ok := firstElem[2].(string)
	if !ok || len(innerStr) < 50 {
		return "", prevText, nil
	}

	var innerArr []interface{}
	if jsonErr := json.Unmarshal([]byte(innerStr), &innerArr); jsonErr != nil {
		return "", prevText, nil
	}

	if len(innerArr) > 4 && innerArr[4] != nil {
		parts, ok := innerArr[4].([]interface{})
		if !ok {
			return "", prevText, nil
		}
		for _, part := range parts {
			partArr, ok := part.([]interface{})
			if !ok || len(partArr) <= 1 || partArr[1] == nil {
				continue
			}
			tList, ok := partArr[1].([]interface{})
			if !ok {
				continue
			}
			for _, t := range tList {
				tStr, ok := t.(string)
				if ok && len(tStr) > len(prevText) {
					deltaText := tStr[len(prevText):]
					cleanedDelta := CleanGeminiText(deltaText, false)
					return cleanedDelta, tStr, nil
				}
			}
		}
	}

	return "", prevText, nil
}
