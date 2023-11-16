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

package kubeclient

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/kubeflow/arena/pkg/apis/config"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

/**
* dry-run creating kubernetes App Info for delete in future
* Exec /usr/local/bin/kubectl, [create --dry-run -f /tmp/values313606961 --namespace default]
**/

func ApplyFromYaml(fileData []byte, namespace string, dryrun bool) (string, error) {

	dynamicClient := config.GetArenaConfiger().GetDynamicClient()
	applyObjects := &bytes.Buffer{}

	// 解析 YAML 文件
	decoder := yaml.NewYAMLOrJSONDecoder(io.NopCloser(bytes.NewReader(fileData)), 4096)

	for {
		var rawObj unstructured.Unstructured
		err := decoder.Decode(&rawObj)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Errorf("decoder: %s error: %v", string(fileData), err)
			return applyObjects.String(), err
		}

		// 获取 GVR
		gvk := rawObj.GroupVersionKind()
		gvr := schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: gvk.Kind}

		dryrunOptions := []string{}
		if dryrun {
			dryrunOptions = append(dryrunOptions, metav1.DryRunAll)
		}

		_, err = dynamicClient.Resource(gvr).Namespace(namespace).Create(context.TODO(), &rawObj, metav1.CreateOptions{DryRun: dryrunOptions})
		if err == nil {
			applyObjects.WriteString(fmt.Sprintf("%s/%s.%s\n", namespace, rawObj.GetKind(), rawObj.GetName()))
			log.Debugf("Resource %s/%s.%s applied successfully namespace: %s\n", namespace, rawObj.GetKind(), rawObj.GetName())
			continue
		}

		if errors.IsAlreadyExists(err) {
			_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(context.TODO(), &rawObj, metav1.UpdateOptions{DryRun: dryrunOptions})
		}
		log.Errorf("apply error: %v", err)
		return applyObjects.String(), err
	}

	return applyObjects.String(), nil
}

func DeleteFromYaml(fileData []byte, namespace string) (string, string, error) {

	dynamicClient := config.GetArenaConfiger().GetDynamicClient()
	okobjs := &bytes.Buffer{}
	failobjs := &bytes.Buffer{}

	// 解析 YAML 文件
	decoder := yaml.NewYAMLOrJSONDecoder(io.NopCloser(bytes.NewReader(fileData)), 4096)
	var err error
	for {
		var rawObj unstructured.Unstructured
		err = decoder.Decode(&rawObj)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Errorf("decoder file: %s error: %v", string(fileData), err)
			return okobjs.String(), failobjs.String(), nil
		}

		// 获取 GVR
		gvk := rawObj.GroupVersionKind()
		gvr := schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: gvk.Kind}

		err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(context.TODO(), rawObj.GetName(), metav1.DeleteOptions{})
		if err == nil {
			okobjs.WriteString(fmt.Sprintf("%s/%s.%s\n", namespace, rawObj.GetKind(), rawObj.GetName()))
			log.Debugf("Resource %s/%s.%s delete successfully namespace: %s\n", namespace, rawObj.GetKind(), rawObj.GetName())
			continue
		}

		if errors.IsNotFound(err) {
			continue
		}
		log.Errorf("delete error: %v", err)
		failobjs.WriteString(fmt.Sprintf("%s/%s.%s\n", namespace, rawObj.GetKind(), rawObj.GetName()))
	}

	return okobjs.String(), failobjs.String(), err
}
