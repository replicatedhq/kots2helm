package builder

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// createChartYAML will create a default Chart.yaml file and put it in the
// root of workspace
func createChartYAML(workspace string, name string, version string) error {
	chart := map[string]interface{}{
		"apiVersion": "v2",
		"name":       name,
		"version":    version,
	}

	rendered, err := yaml.Marshal(chart)
	if err != nil {
		return errors.Wrap(err, "failed to marshal chart")
	}

	fileName := filepath.Join(workspace, "Chart.yaml")

	if err := os.WriteFile(fileName, rendered, 0644); err != nil {
		return errors.Wrap(err, "failed to write chart.yaml")
	}

	return nil
}
