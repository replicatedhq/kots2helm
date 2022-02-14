package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/replicatedhq/kots2helm/pkg/builder"
	"github.com/replicatedhq/kots2helm/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kots2helm",
		Short: "kots2helm",
		Long:  ``,
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if v.GetString("log-level") == "debug" {
				logger.Info("setting log level to debug")
				logger.SetDebug()
			}

			if err := builder.Build(args[0], v.GetString("name"), v.GetString("version")); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("log-level", "info", "log level")
	cmd.Flags().String("name", "", "name of the helm chart to build")
	cmd.MarkFlagRequired("name")
	cmd.Flags().String("version", "", "version of the helm chart to build")
	cmd.MarkFlagRequired("version")

	cobra.OnInitialize(initConfig)

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("REPLICATED")
	viper.AutomaticEnv()
}
