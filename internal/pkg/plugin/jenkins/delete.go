package jenkins

import (
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/devstream-io/devstream/pkg/util/helm"
	"github.com/devstream-io/devstream/pkg/util/k8s"
	"github.com/devstream-io/devstream/pkg/util/log"
)

// Delete deletes jenkins with provided options.
func Delete(options map[string]interface{}) (bool, error) {
	var opts Options
	if err := mapstructure.Decode(options, &opts); err != nil {
		return false, err
	}

	if errs := validate(&opts); len(errs) != 0 {
		for _, e := range errs {
			log.Errorf("Options error: %s.", e)
		}
		return false, fmt.Errorf("opts are illegal")
	}

	h, err := helm.NewHelm(opts.GetHelmParam())
	if err != nil {
		return false, err
	}

	log.Info("Uninstalling jenkins helm chart ...")
	if err = h.UninstallHelmChartRelease(); err != nil {
		return false, err
	}

	if err := dealWithNsWhenDelete(&opts); err != nil {
		return false, err
	}

	if err = postDelete(); err != nil {
		log.Errorf("Failed to execute the post-delete logic. Error: %s.", err)
		return false, err
	}

	return true, nil
}

func dealWithNsWhenDelete(opts *Options) error {
	if !opts.CreateNamespace {
		return nil
	}

	kubeClient, err := k8s.NewClient()
	if err != nil {
		return err
	}

	return kubeClient.DeleteNamespace(opts.Chart.Namespace)
}
