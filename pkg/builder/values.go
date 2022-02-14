package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"gopkg.in/yaml.v3"
)

// createValuesYAML will convert the config.yaml to a values.yaml and put it in the root
// of workspace. This function
func createValuesYAML(workspace string) error {
	objP, err := getKOTSKind(workspace, "kots.io", "v1beta1", "Config")
	if err != nil {
		return errors.Wrap(err, "failed to get config")
	}
	if objP == nil {
		fmt.Printf("no kots config found\n")
		return nil
	}

	obj := *objP
	kotsConfig := obj.(*kotsv1beta1.Config)
	values := map[string]interface{}{}

	// always present
	values["isKurl"] = false

	for _, configGroup := range kotsConfig.Spec.Groups {
		valuesGroup := map[string]interface{}{}
		for _, configItem := range configGroup.Items {
			valuesGroup[configItem.Name] = configItem.Default
		}

		values[configGroup.Name] = valuesGroup
	}

	rendered, err := yaml.Marshal(values)
	if err != nil {
		return errors.Wrap(err, "failed to marshal values")
	}

	fileName := filepath.Join(workspace, "values.yaml")

	if err := os.WriteFile(fileName, rendered, 0644); err != nil {
		return errors.Wrap(err, "failed to write values.yaml")
	}

	return nil
}
