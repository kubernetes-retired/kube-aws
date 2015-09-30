package cluster

import (
	"regexp"
	"strings"
)

var templateRef *regexp.Regexp

func init() {
	templateRef = regexp.MustCompile(`{{\s*[a-zA-Z0-9]+(\|[a-zA-Z0-9]+)*\s*}}`)
}

// renderTemplate creates an AWS template from a rudimentary templating language
func renderTemplate(tmpl string) map[string]interface{} {
	output := []interface{}{}

	pos := 0

	for {
		// find tag
		loc := templateRef.FindStringIndex(tmpl[pos:])

		// append remaining if no tag found
		if loc == nil {
			output = append(output, tmpl[pos:])

			break
		}

		begin := pos + loc[0]
		end := pos + loc[1]

		// tag minus braces
		variable := strings.TrimSpace(tmpl[begin+2 : end-2])
		// get filters
		variableParts := strings.Split(variable, "|")

		// AWS reference
		var part interface{} = map[string]interface{}{
			"Ref": strings.TrimSpace(variableParts[0]),
		}

		// apply AWS functions
		for i := 1; i < len(variableParts); i++ {
			switch strings.TrimSpace(variableParts[i]) {
			case "base64":
				part = map[string]interface{}{"Fn::Base64": part}
			}
		}

		output = append(output, tmpl[pos:begin])
		output = append(output, part)

		// advance to after tag
		pos = end
	}

	return map[string]interface{}{
		"Fn::Join": []interface{}{"", output},
	}
}
