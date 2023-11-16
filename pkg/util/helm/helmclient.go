// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helm

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kubeflow/arena/pkg/apis/config"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/storage/driver"
)

type HelmClient struct {
	actionConfig *action.Configuration
	namespace    string
}

func NewHelmClient(namespace string) (*HelmClient, error) {

	actionConfig := new(action.Configuration)
	err := actionConfig.Init(config.GetArenaConfiger(), namespace, os.Getenv("HELM_DRIVER"), log.Printf)
	if err != nil {
		return nil, err
	}

	client := &HelmClient{actionConfig: actionConfig, namespace: namespace}

	return client, nil
}

func (h *HelmClient) ToYamlMap(vals interface{}, valsMap map[string]interface{}) error {

	yamlBytes, err := yaml.Marshal(vals)
	if err != nil {
		log.Errorf("yaml.Marshal vals:%v err: %v", vals, err)
		return err
	}

	log.Debugf("after Marshal vals: %s", string(yamlBytes))

	if err := yaml.Unmarshal(yamlBytes, &valsMap); err != nil {
		log.Errorf("yaml.Unmarshal vals:%v err: %v", vals, err)
		return err
	}

	return nil
}

func (h *HelmClient) TemplateRelease(name string, valsMap map[string]interface{}, chartName string, options ...string) (string, error) {

	// TODO use func LoadArchive(in io.Reader) (*chart.Chart, error)
	charts, err := loader.Load(chartName)
	if err != nil {
		log.Errorf("load chart: %s failed, err: %v", chartName, err)
		return "", err
	}

	client := action.NewInstall(h.actionConfig)
	client.ReleaseName = name
	client.Namespace = h.namespace
	client.DryRun = true
	client.Replace = true // Skip the name check

	rel, err := client.Run(charts, valsMap)
	if err != nil {
		log.Errorf("render name: %s chart: %s  failed, err: %v", name, chartName, err)
		return "", err
	}
	//  there is no hook in charts
	return rel.Manifest, nil
}

func (h *HelmClient) InstallRelease(name string, valsMap map[string]interface{}, chartName string, options ...string) error {
	// TODO use func LoadArchive(in io.Reader) (*chart.Chart, error)
	charts, err := loader.Load(chartName)
	if err != nil {
		log.Errorf("load chart: %s failed, err: %v", chartName, err)
		return err
	}

	client := action.NewInstall(h.actionConfig)
	client.ReleaseName = name
	client.Namespace = h.namespace
	client.DryRunOption = "none"
	// client.CreateNamespace = true
	client.Replace = true // Skip the name check

	rel, err := client.Run(charts, valsMap)
	if err != nil {
		log.Errorf("install name: %s chart: %s  failed, err: %v", name, chartName, err)
		return err
	}

	log.Debugf("InstallRelease %s/%s done", rel.Namespace, rel.Name)
	// there is no hook in charts
	return nil
}

func (h *HelmClient) UpdateRelease(name string, valsMap map[string]interface{}, chartName string, options ...string) error {
	// TODO use func LoadArchive(in io.Reader) (*chart.Chart, error)
	charts, err := loader.Load(chartName)
	if err != nil {
		log.Errorf("load chart: %s failed, err: %v", chartName, err)
		return err
	}

	client := action.NewUpgrade(h.actionConfig)
	client.Namespace = h.namespace
	// client.CreateNamespace = true
	client.DryRunOption = "none"

	rel, err := client.Run(name, charts, valsMap)
	if err != nil {
		log.Errorf("update name: %s chart: %s  failed, err: %v", name, chartName, err)
		return err
	}

	log.Debugf("UpdateRelease %s/%s done", rel.Namespace, rel.Name)
	// there is no hook in charts
	return nil
}

func (h *HelmClient) CheckRelease(name string) (bool, error) {
	histClient := action.NewHistory(h.actionConfig)
	histClient.Max = 1
	rel, err := histClient.Run(name)
	if err == driver.ErrReleaseNotFound {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if len(rel) == 0 {
		return false, fmt.Errorf("not found release: %s", name)
	}

	if rel[0].Info != nil {
		bytes, _ := json.Marshal(rel[0].Info)
		log.Debugf("chart:%s install info: %s", name, string(bytes))
	}

	return true, nil
}

func (h *HelmClient) UninstallRelease(name string) error {

	client := action.NewUninstall(h.actionConfig)

	res, err := client.Run(name)

	if err != nil {
		return err
	}

	if res != nil {
		log.Infof("uninstall name:%s info:%s", name, res.Info)
	}

	return nil
}
