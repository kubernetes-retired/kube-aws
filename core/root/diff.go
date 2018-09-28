package root

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aryann/difflib"
	"github.com/mgutz/ansi"
	"math"
	"strings"
)

func diffJson(current, desired string, context int) (string, error) {
	var currentBytes bytes.Buffer
	err := json.Indent(&currentBytes, []byte(current), "", "  ")
	if err != nil {
		return "", err
	}

	var desiredBytes bytes.Buffer
	err = json.Indent(&desiredBytes, []byte(desired), "", "  ")
	if err != nil {
		return "", err
	}

	return diffText(currentBytes.String(), desiredBytes.String(), context)
}

func diffText(current, desired string, context int) (string, error) {
	stackDiffs := difflib.Diff(strings.Split(current, "\n"), strings.Split(desired, "\n"))
	stackDiffOutputs := []string{}
	if context >= 0 {
		distances := calculateDistances(stackDiffs)
		omitting := false
		for i, r := range stackDiffs {
			if distances[i] > context {
				if !omitting {
					stackDiffOutputs = append(stackDiffOutputs, "...")
					omitting = true
				}
			} else {
				omitting = false
				stackDiffOutputs = append(stackDiffOutputs, sprintDiffRecord(r))
			}
		}
	} else {
		for _, r := range stackDiffs {
			stackDiffOutputs = append(stackDiffOutputs, sprintDiffRecord(r))
		}
	}
	return strings.Join(stackDiffOutputs, ""), nil
}

// Calculate distance of every diff-line to the closest change
func calculateDistances(diffs []difflib.DiffRecord) map[int]int {
	distances := map[int]int{}

	// Iterate forwards through diffs, set 'distance' based on closest 'change' before this line
	change := -1
	for i, diff := range diffs {
		if diff.Delta != difflib.Common {
			change = i
		}
		distance := math.MaxInt32
		if change != -1 {
			distance = i - change
		}
		distances[i] = distance
	}

	// Iterate backwards through diffs, reduce 'distance' based on closest 'change' after this line
	change = -1
	for i := len(diffs) - 1; i >= 0; i-- {
		diff := diffs[i]
		if diff.Delta != difflib.Common {
			change = i
		}
		if change != -1 {
			distance := change - i
			if distance < distances[i] {
				distances[i] = distance
			}
		}
	}

	return distances
}

func sprintDiffRecord(diff difflib.DiffRecord) string {
	text := diff.Payload

	var res string
	switch diff.Delta {
	case difflib.RightOnly:
		res = fmt.Sprintf("%s\n", ansi.Color("+ "+text, "green"))
	case difflib.LeftOnly:
		res = fmt.Sprintf("%s\n", ansi.Color("- "+text, "red"))
	case difflib.Common:
		res = fmt.Sprintf("%s\n", "  "+text)
	}
	return res
}
