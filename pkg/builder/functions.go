package builder

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

// replaceKOTSTemplatesWithHelmTemplates handles converting kots templates to helm templates
func replaceKOTSTemplatesWithHelmTemplates(workspace string) error {
	objP, err := getKOTSKind(workspace, "kots.io", "v1beta1", "Config")
	if err != nil {
		return errors.Wrap(err, "failed to get config")
	}
	if objP == nil {
		// there isn't a kots config
		// TODO: should we error here?
		return nil // no maybe its plain k8s
	}

	obj := *objP
	kotsConfig := obj.(*kotsv1beta1.Config)

	remainingKotsTemplateFunctionCount := false

	err = filepath.Walk(filepath.Join(workspace, "templates"),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
				return err
			}

			if info.IsDir() {
				return nil
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			isKots, err := isKOTSManifest(content)
			if err != nil {
				return err
			}
			if isKots {
				return nil
			}

			c, err := replaceWhenAndExcludeAnnotations(content, kotsConfig)
			if err != nil {
				return err
			}
			content = c

			helmContent, err := helmify(content, kotsConfig)
			if err != nil {
				return errors.Wrap(err, "failed to helmify")
			}

			// assert that there are no {{repl or repl{{ templates left.
			// if there are, we need to fail the build
			hasTemplateFunctions, err := numKotsTemplateFunctions(helmContent)
			if err != nil {
				return errors.Wrap(err, "failed to check for kots template functions")
			}

			if hasTemplateFunctions > 0 {
				fmt.Printf("%s has %d kots template functions\n", path, hasTemplateFunctions)
				remainingKotsTemplateFunctionCount = true
			}

			if err := ioutil.WriteFile(path, helmContent, info.Mode()); err != nil {
				return err
			}
			return nil
		})

	if err != nil {
		return errors.Wrap(err, "failed to walk workspace")
	}

	if remainingKotsTemplateFunctionCount {
		return errors.New("kots template functions remain")
	}

	return nil
}

func helmify(content []byte, kotsConfig *kotsv1beta1.Config) ([]byte, error) {
	// we can ignore kots manifests since they will be deleted

	// content will be updated and resaved at the end of the function

	// ConfigOption
	c, err := replaceConfigOption(content, kotsConfig)
	if err != nil {
		return nil, err
	}
	content = c

	// ConfigOptionData

	// ConfigOptionFilename

	// ConfigOptionEquals
	c, err = replaceConfigOptionEquals(content, kotsConfig)
	if err != nil {
		return nil, err
	}
	content = c

	// ConfigOptionNotEquals

	// Namespace
	c, err = replaceNamespace(content)
	if err != nil {
		return nil, err
	}
	content = c

	// IsKurl
	c, err = replaceIsKurl(content)
	if err != nil {
		return nil, err
	}
	content = c

	// if and conditional
	c, err = replaceIfAndConditional(content)
	if err != nil {
		return nil, err
	}
	content = c

	return content, nil
}

func replaceIfAndConditional(content []byte) ([]byte, error) {
	delimiters := map[string]string{
		`(?:{{repl\s+if)(?:\s?)`:   "{{ if ",
		`(?:repl{{\s+if)(?:\s?)`:   "{{ if ",
		`(?:{{repl\s+else)(?:\s?)`: "{{ else ",
		`(?:repl{{\s+else)(?:\s?)`: "{{ else ",
		`(?:{{repl\s+end)(?:\s?)`:  "{{ end ",
		`(?:repl{{\s+end)(?:\s?)`:  "{{ end ",
	}

	updatedContent := string(content)

	for delimiter, replace := range delimiters {
		r := regexp.MustCompile(delimiter)
		regexMatch := r.FindAllStringSubmatch(string(content), -1)
		for _, result := range regexMatch {
			updatedContent = strings.ReplaceAll(updatedContent, result[0], replace)
		}
	}

	return []byte(updatedContent), nil
}

// replaceIsKurl IsKurl
func replaceIsKurl(content []byte) ([]byte, error) {
	delimiters := map[string]string{
		`(?:{{repl\s+IsKurl)(?:\s?)`:       `{{ .Values.isKurl `,
		`(?:repl{{\s+IsKurl)(?:\s?)`:       `{{ .Values.isKurl `,
		`(?:{{repl\s+not\s+IsKurl)(?:\s?)`: `{{ not .Values.isKurl `,
		`(?:repl{{\s+not\s+IsKurl)(?:\s?)`: `{{ not .Values.isKurl `,
	}

	updatedContent := string(content)

	for delimiter, value := range delimiters {
		r := regexp.MustCompile(delimiter)
		regexMatch := r.FindAllStringSubmatch(string(content), -1)
		for _, result := range regexMatch {
			updatedContent = strings.ReplaceAll(updatedContent, result[0], value)
		}
	}

	return []byte(updatedContent), nil
}

func replaceNamespace(content []byte) ([]byte, error) {
	delimiters := []string{
		`(?:{{repl\s+Namespace)(?:\s?}})`,
		`(?:repl{{\s+Namespace)(?:\s?}})`,
	}

	updatedContent := string(content)

	for _, delimiter := range delimiters {
		r := regexp.MustCompile(delimiter)
		regexMatch := r.FindAllStringSubmatch(string(content), -1)
		for _, result := range regexMatch {
			updatedContent = strings.ReplaceAll(updatedContent, result[0], `{{ .Release.Namespace }}`)
		}
	}

	return []byte(updatedContent), nil
}

func replaceConfigOptionEquals(content []byte, kotsConfig *kotsv1beta1.Config) ([]byte, error) {
	// this is a supoer basic implementation for now
	delimiters := []string{
		`(?:{{repl\s+ConfigOptionEquals\s+\")(?P<Item>.*)(?:\"\s+\")(?P<Value>.*)(?:\"\s?}})`,
		`(?:repl{{\s+ConfigOptionEquals\s+\")(?P<Item>.*)(?:\"\s+\")(?P<Value>.*)(?:\"\s?}})`,
		// TODO " vs ' vs ` and more"
	}

	updatedContent := string(content)

	for _, delimiter := range delimiters {
		r := regexp.MustCompile(delimiter)
		regexMatch := r.FindAllStringSubmatch(string(content), -1)
		for _, result := range regexMatch {
			valuesType, valuesPath, err := getValuesTypeAndPathForConfigItem(result[1], kotsConfig)
			if err != nil {
				// we don't error here, it will catch it later if the function remains
				// in the yaml
				continue
			}

			// TODO this is not the only use of ConfigOptionEquals
			switch valuesType {
			case "string", "password":
				updatedContent = strings.ReplaceAll(updatedContent, result[0], fmt.Sprintf(`{{ if eq .Values.%s %q }}true{{ else }}false{{ end }}`, valuesPath, result[2]))
			case "bool":
				v, err := strconv.ParseBool(result[2])
				if err != nil {
					return nil, errors.Wrap(err, "failed to parse bool")
				}
				updatedContent = strings.ReplaceAll(updatedContent, result[0], fmt.Sprintf(`{{ if eq .Values.%s %t }}true{{ else }}false{{ end }}`, valuesPath, v))
			}

		}
	}

	return []byte(updatedContent), nil
}

func replaceWhenAndExcludeAnnotations(content []byte, kotsConfig *kotsv1beta1.Config) ([]byte, error) {
	annotations, err := getAnnotations(content)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations")
	}

	for k, v := range annotations {
		if k == "kots.io/when" {
			// convert the value to a helm template
			helmed, err := helmify([]byte(v), kotsConfig)
			if err != nil {
				return nil, errors.Wrap(err, "failed to helmify")
			}

			// we remove the annotation, even though it's harmless because
			// when we leave it, we can't detect it in our "any templates left?" check
			updatedAnnotations := map[string]string{}
			for otherK, otherV := range annotations {
				if otherK != "kots.io/when" {
					updatedAnnotations[otherK] = otherV
				}
			}

			withoutWhen, err := withAnnotations(content, updatedAnnotations)
			if err != nil {
				return nil, errors.Wrap(err, "failed to remove when annotation")
			}
			return []byte(fmt.Sprintf(`{{ if %s }}
%s
{{ end }}`, string(helmed), strings.TrimSpace(string(withoutWhen)))), nil
		} else if k == "kots.io/exclude" {
			// convert the value to a helm template
			helmed, err := helmify([]byte(v), kotsConfig)
			if err != nil {
				return nil, errors.Wrap(err, "failed to helmify")
			}

			// when we leave it, we can't detect it in our "any templates left?" check
			updatedAnnotations := map[string]string{}
			for otherK, otherV := range annotations {
				if otherK != "kots.io/exclude" {
					updatedAnnotations[otherK] = otherV
				}
			}

			withoutExclude, err := withAnnotations(content, updatedAnnotations)
			if err != nil {
				return nil, errors.Wrap(err, "failed to remove excude annotation")
			}
			return []byte(fmt.Sprintf(`{{ if ne %s }}
%s
{{ end }}`, string(helmed), strings.TrimSpace(string(withoutExclude)))), nil
		}
	}

	return content, nil
}

func replaceConfigOption(content []byte, kotsConfig *kotsv1beta1.Config) ([]byte, error) {
	// this is a supoer basic implementation for now
	delimiters := map[string]string{
		`(?:{{repl\s+ConfigOption\s+\")(?P<Item>.*)(?:\"\s?}})`:  `{{ .Values.%s }}`,
		`(?:repl{{\s+ConfigOption\s+\")(?P<Item>.*)(?:\"\s?}})`:  `{{ .Values.%s }}`,
		"(?:{{repl\\s+ConfigOption\\s+`)(?P<Item>.*)(?:`\\s?}})": `{{ .Values.%s }}`,
		"(?:repl{{\\s+ConfigOption\\s+`)(?P<Item>.*)(?:`\\s?}})": `{{ .Values.%s }}`,

		// after those ^^ are replaced, these will be safer (without the end delimiter)
		`(?:{{repl\s+ConfigOption\s+\")(?P<Item>.*)(?:\"\s?)`:  `{{ .Values.%s `,
		`(?:repl{{\s+ConfigOption\s+\")(?P<Item>.*)(?:\"\s?)`:  `{{ .Values.%s `,
		"(?:{{repl\\s+ConfigOption\\s+`)(?P<Item>.*)(?:`\\s?)": `{{ .Values.%s `,
		"(?:repl{{\\s+ConfigOption\\s+`)(?P<Item>.*)(?:`\\s?)": `{{ .Values.%s `,
	}

	updatedContent := string(content)

	for delimiter, value := range delimiters {
		r := regexp.MustCompile(delimiter)
		regexMatch := r.FindAllStringSubmatch(string(content), -1)
		for _, result := range regexMatch {
			_, valuesPath, err := getValuesTypeAndPathForConfigItem(result[1], kotsConfig)
			if err != nil {
				// we don't error here, it will catch it later if the function remains
				// in the yaml
				continue
			}

			updatedContent = strings.ReplaceAll(updatedContent, result[0], fmt.Sprintf(value, valuesPath))
		}

	}

	return []byte(updatedContent), nil
}

func getValuesTypeAndPathForConfigItem(itemName string, kotsConfig *kotsv1beta1.Config) (string, string, error) {
	for _, group := range kotsConfig.Spec.Groups {
		for _, item := range group.Items {
			if item.Name == itemName {
				return item.Type, fmt.Sprintf("%s.%s", group.Name, item.Name), nil
			}
		}
	}

	return "", "", errors.Errorf("failed to find config item %s", itemName)
}

func numKotsTemplateFunctions(content []byte) (int, error) {
	numReplFns := 0

	// {{repl
	numReplFns += len(regexp.MustCompile(`{{repl\s+`).FindAllString(string(content), -1))

	// repl{{
	numReplFns += len(regexp.MustCompile(`repl{{\s+`).FindAllString(string(content), -1))

	return numReplFns, nil
}
