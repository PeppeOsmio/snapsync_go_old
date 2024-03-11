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
		configsDir, err := cmd.Flags().GetString("config-dir")
		if err != nil {
			slog.Error("can't get config-dir flag")
			return
		}
		expandVars, err := cmd.Flags().GetBool("expand-vars")
		if err != nil {
			slog.Error("can't get expand-vars flag")
			return
		}
		config, err := configs.LoadConfig(configsDir, expandVars)
		if err != nil {
			slog.Error("can't get " + configsDir + ": " + err.Error())
			return
		}

		runOnce, err := cmd.Flags().GetStringArray("run-once")
		if err != nil {
			slog.Error(fmt.Sprintf("can't get run-once flag: %s", err.Error()))
			return
		}

		snapshotsConfigs, err := configs.LoadSnapshotsConfigs(config.SnapshotsConfigsDir, expandVars)
		if err != nil {
			slog.Error("Can't get snapshots configs in " + configsDir + ": " + err.Error())
		}

		snapshotTask := func(snapshotConfig *structs.SnapshotConfig) {
			snapshotErr := snapshots.ExecuteSnapshot(config, snapshotConfig)
			if snapshotErr != nil {
				slog.Error(fmt.Sprintf("[%s] can't execute snapshot: %s", snapshotConfig.SnapshotName, snapshotErr.Error()))
			}
		}

		if len(runOnce) > 0 {
			for _, snapshotToRun := range runOnce {
				var sc *structs.SnapshotConfig
				for _, snapshotConfig := range snapshotsConfigs {
					if snapshotToRun == snapshotConfig.SnapshotName {
						sc = snapshotConfig
						break
					}
				}
				if sc != nil {
					snapshotTask(sc)
				} else {
					slog.Error(fmt.Sprintf("there is not snapshot named %s", runOnce))
				}
			}
			return
		}

		snapshotsConfigsToSchedule := []*structs.SnapshotConfig{}
		for _, snapshotConfig := range snapshotsConfigs {
			if len(snapshotConfig.Cron) == 0 {
				continue
			}
			snapshotsConfigsToSchedule = append(snapshotsConfigsToSchedule, snapshotConfig)
		}
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
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("config-dir", configs.GetDefaultConfigsDir(), "Directory where config.yml is stored")
	rootCmd.PersistentFlags().Bool("expand-vars", true, "Expand env variables in the config files")
	rootCmd.Flags().StringArray("run-once", []string{}, "Run these snapshots once")
}
