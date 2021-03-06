package builder

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/plus3it/gorecurcopy"
	"github.com/replicatedhq/kots2helm/pkg/logger"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
)

// Build will create a helm chart from the given input dir
func Build(inputDir string, name string, version string) error {

	// create a temp dir with a copy of the workspace so we can edit
	workspace, err := ioutil.TempDir("", "helm")
	if err != nil {
		return err
	}

	wasSuccessful := false
	defer func() {
		if wasSuccessful {
			os.RemoveAll(workspace)
			return
		}

		fmt.Printf("some templates did not convert. the work is left in %s\n", workspace)
	}()

	if err := os.MkdirAll(filepath.Join(workspace, "templates"), 0755); err != nil {
		return err
	}

	if err := gorecurcopy.CopyDirectory(inputDir, filepath.Join(workspace, "templates")); err != nil {
		return err
	}

	// TODO: copy depenencies and existing helm charts

	if err := createValuesYAML(workspace); err != nil {
		return err
	}

	if err := createChartYAML(workspace, name, version); err != nil {
		return err
	}

	remainingKOTSTemplateFunctionsMap, err := replaceKOTSTemplatesWithHelmTemplates(workspace)
	if err != nil {
		return err
	}

	// if err := replaceStaticImagesWithTemplates(build, workspace); err != nil {
	// 	return err
	// }

	if err := removeKOTSManifests(workspace); err != nil {
		return err
	}

	archiveFile, err := packageHelmChart(workspace)
	if err != nil {
		return err
	}

	wasSuccessful = len(remainingKOTSTemplateFunctionsMap) == 0

	if !wasSuccessful {
		fmt.Println("The following files have template functions that could not be converted:")
		for path, count := range remainingKOTSTemplateFunctionsMap {
			fmt.Printf("%s: %d\n", path, count)
		}
	}

	fmt.Printf("chart is at %s\n", archiveFile)

	// if err := build.publishHelmChart(archiveFile, r); err != nil {
	// 	buildError = errors.Wrap(err, "failed to publish helm chart")
	// 	return
	// }

	return nil
}

// removeKOTSManifests will remove all kots manifests from the root of workspace
// this should be done last as other methods in build will rely on these to exist
func removeKOTSManifests(workspace string) error {
	err := filepath.Walk(filepath.Join(workspace, "templates"),
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			yamlDoc, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			shouldDelete, err := isKOTSManifest(yamlDoc)
			if err != nil {
				return err
			}

			if shouldDelete {
				logger.Verbosef("removing %s because it's a KOTS manifest", path)
				if err := os.Remove(path); err != nil {
					return errors.Wrap(err, "failed to remove file")
				}
			}

			return nil
		})

	if err != nil {
		return errors.Wrap(err, "failed to remove kots manifests")
	}

	return nil
}

func packageHelmChart(workspace string) (string, error) {
	client := action.NewPackage()
	valueOpts := &values.Options{}

	settings := cli.New()
	providers := getter.All(settings)

	vals, err := valueOpts.MergeValues(providers)
	if err != nil {
		return "", errors.Wrap(err, "failed to merge values")
	}

	p, err := client.Run(workspace, vals)
	if err != nil {
		return "", errors.Wrap(err, "failed to package helm chart")
	}

	return p, nil
}
