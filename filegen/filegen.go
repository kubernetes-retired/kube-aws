package filegen

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

func CreateFileFromTemplate(outputFilePath string, templateOpts interface{}, fileTemplate []byte) error {
	// Render the default cluster config.
	cfgTemplate, err := template.New("cluster.yaml").Parse(string(fileTemplate))
	if err != nil {
		return fmt.Errorf("Error parsing default config template: %v", err)
	}

	dir := filepath.Dir(outputFilePath)

	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("Error creating directory: %v", err)
		}
	}

	out, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("Error opening %s : %v", outputFilePath, err)
	}
	defer out.Close()
	if err := cfgTemplate.Execute(out, templateOpts); err != nil {
		return fmt.Errorf("Error exec-ing default config template: %v", err)
	}
	return nil
}
