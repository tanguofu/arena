package commands

import (
	"encoding/json"
	"fmt"

	"github.com/kubeflow/arena/pkg/apis/arenaclient"
	"github.com/kubeflow/arena/pkg/apis/config"
	"github.com/kubeflow/arena/pkg/apis/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewWhoamiCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "whoami",
		Short: "Display current user information.",
		Long:  "Display current user information.",
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := arenaclient.NewArenaClient(types.ArenaClientArgs{
				Kubeconfig:     viper.GetString("config"),
				LogLevel:       viper.GetString("loglevel"),
				Namespace:      viper.GetString("namespace"),
				ArenaNamespace: viper.GetString("arena-namespace"),
				IsDaemonMode:   false,
			})
			if err != nil {
				return fmt.Errorf("failed to create arena client: %v", err)
			}
			user := config.GetArenaConfiger().GetUser()
			cluster := config.GetArenaConfiger().GetCluster()
			d, err := json.Marshal(struct {
				Name        string
				Id          string
				Account     string
				Group       string
				IsAdminUser bool
				Server      string
				ClusterName string
			}{
				Name:        user.GetName(),
				Id:          user.GetId(),
				Account:     user.GetAccount(),
				Group:       user.GetGroup(),
				IsAdminUser: config.GetArenaConfiger().IsAdminUser(),
				Server:      cluster.GetServer(),
				ClusterName: cluster.GetName(),
			})
			if err != nil {
				return err
			}
			fmt.Println(string(d))
			return nil
		},
	}
	return command
}
