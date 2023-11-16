package cron

import (
	"github.com/kubeflow/arena/pkg/apis/types"
	"github.com/kubeflow/arena/pkg/util"
	"github.com/kubeflow/arena/pkg/workflow"
	log "github.com/sirupsen/logrus"
)

func SubmitCronTFJob(namespace string, submitArgs *types.CronTFJobArgs) (err error) {
	cronTFJobChart := util.GetChartsFolder() + "/cron-tfjob"

	err = workflow.SubmitJobByHelm(submitArgs.Name, string(types.CronTFTrainingJob), namespace, submitArgs, cronTFJobChart, submitArgs.HelmOptions...)
	if err != nil {
		return err
	}
	log.Infof("The cron tfjob %s has been submitted successfully", submitArgs.Name)
	log.Infof("You can run `arena cron get %s` to check the cron status", submitArgs.Name)

	return nil
}
