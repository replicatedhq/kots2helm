package builder

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	KOTSConfigGVK = "kots.io/v1beta1/Config"
)

type OverlySimpleGVK struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

type OverlySimpleGVKWithAnnotations struct {
	APIVersion string              `yaml:"apiVersion"`
	Kind       string              `yaml:"kind"`
	Metadata   OverySimplyMetadata `yaml:"metadata"`
}

type OverySimplyMetadata struct {
	Annotations map[string]string `yaml:"annotations"`
}

func isKOTSManifest(content []byte) (bool, error) {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return false, nil // it's not a kots manifest
	}

	if strings.HasPrefix(o.APIVersion, "kots.io") {
		return true, nil
	}
	if strings.HasPrefix(o.APIVersion, "troubleshoot.replicated.com") {
		return true, nil
	}
	if o.APIVersion == "app.k8s.io/v1beta1" && o.Kind == "Application" {
		return true, nil
	}

	// TODO what else?
	return false, nil
}

func getGVK(content []byte) (string, error) {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal yaml")
	}

	return fmt.Sprintf("%s/%s", o.APIVersion, o.Kind), nil
}

func getAnnotations(content []byte) (map[string]string, error) {
	o := OverlySimpleGVKWithAnnotations{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal yaml")
	}

	return o.Metadata.Annotations, nil
}

func withAnnotations(content []byte, annotations map[string]string) ([]byte, error) {
	y := map[string]interface{}{}
	if err := yaml.Unmarshal(content, &y); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal yaml")
	}

	metadata := y["metadata"].(map[interface{}]interface{})
	metadata["annotations"] = annotations
	y["metadata"] = metadata

	remarshaled, err := yaml.Marshal(y)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal yaml")
	}

	return remarshaled, nil
}
