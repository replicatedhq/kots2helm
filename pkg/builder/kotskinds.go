package builder

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

// getKOTSKind will find the requested kots kind in workspace
func getKOTSKind(workspace string, g, v, k string) (*runtime.Object, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	var foundObj *runtime.Object
	err := filepath.Walk(workspace,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			yamlDoc, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			// be careful, some apps have binaries and non-yaml manifests mixed in

			gvk, err := getGVK(yamlDoc)
			if err != nil {
				return nil
			}

			if gvk == KOTSConfigGVK {
				// decode it properly using a scheme
				o, _, err := decode(yamlDoc, nil, nil)
				if err != nil {
					fmt.Printf("failed to decode yaml: %s\n", err)
					return err
				}

				foundObj = &o
				return nil
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	return foundObj, nil
}
