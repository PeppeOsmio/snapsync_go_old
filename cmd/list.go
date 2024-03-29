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

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the snapshots",
	Long:  `List the snapshots`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configsDir, err := cmd.Flags().GetString("config-dir")
		if err != nil {
			slog.Error("can 't get configs-dir flag")
			return
		}
		expandVars, err := cmd.Flags().GetBool("expand-vars")
		if err != nil {
			slog.Error("can 't get expand-vars flag")
			return
		}
		config, err := configs.LoadConfig(configsDir, expandVars)
		if err != nil {
			slog.Error("can't get " + configsDir + ": " + err.Error())
			return
		}
		snapshotToList := args[0]
		snapshotsInfo, err := snapshots.GetSnapshotsInfo(config.SnapshotsConfigsDir, expandVars, snapshotToList)
		if err != nil {
			slog.Error("Can't get snapshots of snapshot " + snapshotToList + ": " + err.Error())
			return
		}
		for _, snapshotInfo := range snapshotsInfo {
			size, err := snapshotInfo.Size()
			sizeStr := ""
			if err != nil {
				sizeStr = fmt.Sprintf("can't evaluate snapshot size: %s", err.Error())
			} else {
				sizeStr = utils.HumanReadableSize(size)
			}
			fmt.Printf("%s, size: %s\n", snapshotInfo.CompactName(), sizeStr)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
