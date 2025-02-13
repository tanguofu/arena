package model

import (
	"fmt"

	"github.com/kubeflow/arena/pkg/apis/types"
	"github.com/kubeflow/arena/pkg/util"
	"github.com/kubeflow/arena/pkg/workflow"
	log "github.com/sirupsen/logrus"
)

func SubmitModelOptimizeJob(namespace string, args *types.ModelOptimizeArgs) error {
	args.Namespace = namespace

	if args.Command == "" {
		args.Command = fmt.Sprintf("python easy_inference/main.py optimize --optimizer=%s --model-config-file=%s --export-path=%s",
			args.Optimizer, args.ModelConfigFile, args.ExportPath)
	}

	modelJobChart := util.GetChartsFolder() + "/modeljob"
	err := workflow.SubmitJobByHelm(args.Name, string(types.ModelOptimizeJob), namespace, args, modelJobChart, args.HelmOptions...)
	if err != nil {
		return err
	}
	log.Infof("The model optimize job %s has been submitted successfully", args.Name)
	log.Infof("You can run `arena model get %s` to check the job status", args.Name)
	return nil
}
