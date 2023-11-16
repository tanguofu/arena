package workflow

import (
	"fmt"
	"os"

	"github.com/kubeflow/arena/pkg/util/helm"
	log "github.com/sirupsen/logrus"
)

func SubmitJobByHelm(name string, trainingType string, namespace string, values interface{}, chart string, options ...string) error {
	h, err := helm.NewHelmClient(namespace)
	if err != nil {
		log.Errorf("init helm client failed, err: %v", err)
		return err
	}

	chartName := fmt.Sprintf("%s-%s", name, trainingType)
	exist, err := h.CheckRelease(chartName)
	if err != nil {
		log.Errorf("check release: %s/%s failed, err: %v", namespace, chartName, err)
		return err
	}

	valsMap, err := h.ToYamlMap(values)
	if err != nil {
		log.Errorf("parse value failed, %+v", values)
		return err
	}

	templates, err := h.TemplateRelease(chartName, valsMap, chart, options...)
	if err != nil {
		log.Errorf("template release: %s/%s failed, err: %v", namespace, chartName, err)
		return err
	}

	if file, err := os.CreateTemp("/tmp", fmt.Sprintf("%s-%s-*.yaml", namespace, chartName)); err == nil {
		log.Infof("save %s/%s template yaml into: %s", namespace, chartName, file.Name())
		file.WriteString(templates)
		file.Close()
	}

	if !exist {
		return h.InstallRelease(chartName, valsMap, chart, options...)
	}

	return h.UpdateRelease(chartName, valsMap, chart, options...)
}

func DeleteJobByHelm(name, namespace, trainingType string) error {
	h, err := helm.NewHelmClient(namespace)
	if err != nil {
		log.Errorf("init helm client failed, err: %v", err)
		return err
	}

	chartName := fmt.Sprintf("%s-%s", name, trainingType)
	ok, err := h.CheckRelease(chartName)
	if err != nil {
		log.Errorf("check release: %s/%s failed, err: %v", namespace, chartName, err)
		return err
	}

	if !ok {
		log.Warnf("release is not install, name: %s/%s", namespace, chartName)
		return nil
	}

	err = h.UninstallRelease(chartName)
	return err
}
