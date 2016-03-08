package config

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/coreos/coreos-cloudinit/config/validate"
)

func TestCloudConfigTemplating(t *testing.T) {
	cluster, err := clusterFromBytes([]byte(MinimalConfigYaml))
	if err != nil {
		t.Fatalf("Unable to load cluster config: %v", err)
	}
	assets, err := cluster.NewTLSAssets()
	if err != nil {
		t.Fatalf("Error generating default assets: %v", err)
	}

	cfg, err := cluster.Config()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	compactAssets, err := assets.Compact(cfg)
	if err != nil {
		t.Fatalf("failed to compress TLS assets: %v", err)
	}

	cfg.TLSConfig = compactAssets

	for _, cloudTemplate := range []struct {
		Name     string
		Template []byte
	}{
		{
			Name:     "CloudConfigWorker",
			Template: CloudConfigWorker,
		},
		{
			Name:     "CloudConfigController",
			Template: CloudConfigController,
		},
	} {
		tmpl, err := template.New(cloudTemplate.Name).Parse(string(cloudTemplate.Template))
		if err != nil {
			t.Errorf("Error loading template %s : %v", cloudTemplate.Name, err)
			continue
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, cfg); err != nil {
			t.Errorf("Error excuting template %s : %v", cloudTemplate.Name, err)
			continue
		}

		report, err := validate.Validate(buf.Bytes())

		if err != nil {
			t.Errorf("cloud-config %s could not be parsed: %v", cloudTemplate.Name, err)
			continue
		}

		for _, entry := range report.Entries() {
			t.Errorf("%s: %+v", cloudTemplate.Name, entry)
		}
	}
}
