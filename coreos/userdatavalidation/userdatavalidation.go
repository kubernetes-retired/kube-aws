package userdatavalidation

import (
	"fmt"
	"github.com/coreos/coreos-cloudinit/config/validate"
	"strings"
)

type Entry struct {
	Name    string
	Content string
}

func Execute(entries []Entry) error {
	errors := []string{}

	for _, userData := range entries {
		report, err := validate.Validate([]byte(userData.Content))

		if err != nil {
			errors = append(
				errors,
				fmt.Sprintf("cloud-config %s could not be parsed: %v",
					userData.Name,
					err,
				),
			)
			continue
		}

		for _, entry := range report.Entries() {
			errors = append(errors, fmt.Sprintf("%s: %+v", userData.Name, entry))
		}
	}

	if len(errors) > 0 {
		reportString := strings.Join(errors, "\n")
		return fmt.Errorf("cloud-config validation errors:\n%s\n", reportString)
	}

	return nil
}
