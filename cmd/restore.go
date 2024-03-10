/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log/slog"
	"snapsync/configs"
	"snapsync/snapshots"
	"snapsync/utils"

	"github.com/spf13/cobra"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a snapshot",
	Long:  `Restore a snapshot`,
	Args:  cobra.ExactArgs(1),
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

		snapshotToRestore := args[0]

		snapshotName, number, err := utils.GetInfoFromSnapshotBasePath(snapshotToRestore)
		if err != nil {
			slog.Error(fmt.Sprintf("can't get snapshot info: %s", err.Error()))
			return
		}

		snapshotConfig, err := configs.GetSnapshotConfigByName(configsDir, expandVars, snapshotName)
		if err != nil {
			slog.Error("An error occurred: " + err.Error())
			return
		}

		err = snapshots.RestoreSnapshot(config, number, snapshotConfig)
		if err != nil {
			slog.Error("an error occurred while restoring the snapshot: " + err.Error())
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}
