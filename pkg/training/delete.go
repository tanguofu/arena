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

package training

import (
	"fmt"

	"github.com/kubeflow/arena/pkg/apis/training"
	"github.com/kubeflow/arena/pkg/apis/types"
	"github.com/kubeflow/arena/pkg/workflow"
	log "github.com/sirupsen/logrus"
)

func DeleteTrainingJob(jobName, namespace string, jobType types.TrainingJobType) error {

	// 1. jobName with type
	jobPrefix, typeName := training.SplitJobName(jobName)
	for trainingType, info := range types.TrainingTypeMap {
		if typeName == string(trainingType) || typeName == info.Alias || typeName == info.Shorthand {
			log.Infof("delete job namespace:%s name:%s type:%s", namespace, jobName, typeName)
			return workflow.DeleteJobByHelm(jobPrefix, namespace, string(typeName))
		}
	}
	// 2 check helm list

	// 2. Handle training jobs created by arena
	trainingTypes, err := getTrainingTypes(jobName, namespace)
	if err != nil {
		return err
	}
	if len(trainingTypes) == 0 {
		return fmt.Errorf("not found job namespace:%s name:%s", namespace, jobName)
	}

	log.Infof("delete job namespace:%s name:%s type:%s", namespace, jobName, trainingTypes[0])
	err = workflow.DeleteJobByHelm(jobName, namespace, trainingTypes[0])
	if err != nil {
		return err
	}
	log.Infof("The training job %s has been deleted successfully", jobName)
	// (TODO: cheyang)3. Handle training jobs created by others, to implement

	return nil
}
