/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"snapsync/configs"
	"snapsync/snapshots"
	"snapsync/structs"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "snapsync",
	Short: "Snapsync is tool that performs snapshots of directories using rsync and hard links to use less space.",
	Long:  `Snapsync is tool that performs snapshots of directories using rsync and hard links to use less space.`,
	Run: func(cmd *cobra.Command, args []string) {
		configsDir, err := cmd.Flags().GetString("configs-dir")
		if err != nil {
			slog.Error("can 't get configs-dir flag")
		}
		expandVars, err := cmd.Flags().GetBool("expand-vars")
		if err != nil {
			slog.Error("can 't get expand-vars flag")
		}
		config, err := configs.LoadConfig(configsDir, expandVars)
		if err != nil {
			slog.Error("can't get " + configsDir + ": " + err.Error())
			return
		}

		snapshotsConfigs, err := configs.LoadSnapshotsConfigs(configsDir, expandVars)
		if err != nil {
			slog.Error("Can't get snapshots configs in " + configsDir + ": " + err.Error())
		}

		snapshotTask := func(snapshotConfig *structs.SnapshotConfig) {
			snapshotErr := snapshots.ExecuteSnapshot(config, snapshotConfig)
			if snapshotErr != nil {
				slog.Error(fmt.Sprintf("[%s] can't execute snapshot: %s", snapshotConfig.SnapshotName, err.Error()))
			}
		}
		snapshotsConfigsToSchedule := []*structs.SnapshotConfig{}
		for _, snapshotConfig := range snapshotsConfigs {

			if len(snapshotConfig.Cron) > 0 {
				snapshotsConfigsToSchedule = append(snapshotsConfigsToSchedule, snapshotConfig)

			} else {
				snapshotTask(snapshotConfig)
			}
		}
		if len(snapshotsConfigsToSchedule) > 0 {
			scheduler, err := gocron.NewScheduler()
			for _, snapshotConfig := range snapshotsConfigsToSchedule {
				_, err := scheduler.NewJob(
					gocron.CronJob(snapshotConfig.Cron, false),
					gocron.NewTask(
						snapshotTask,
						snapshotConfig,
					),
				)
				if err != nil {
					slog.Error("Can't add cron job for snapshot " + snapshotConfig.SnapshotName + ". Cron string is " + snapshotConfig.Cron)
					return
				}
				slog.Info(fmt.Sprintf("[%s] scheduled with cron %s", snapshotConfig.SnapshotName, snapshotConfig.Cron))
			}
			if err != nil {
				slog.Error("can't create scheduler.")
				return
			}
			scheduler.Start()
			for {
				time.Sleep(1 * time.Second)
			}
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	defaultConfigsPath, err := configs.GetDefaultConfigsDir()
	if err != nil {
		slog.Error("Can't get default config path: " + err.Error())
		return
	}
	rootCmd.PersistentFlags().String("configs-dir", defaultConfigsPath, "Directory where the configs are stored")
	rootCmd.PersistentFlags().Bool("expand-vars", true, "Expand env variables in the config files")
}
