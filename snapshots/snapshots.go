package snapshots

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"snapsync/configs"
	"snapsync/structs"
	"snapsync/utils"
	"strconv"
	"time"
)

func getRsyncDirsCommand(config *structs.Config, srcDir string, dstDir string, excludes []string) string {
	rsyncExecutable := "rsync"
	if len(config.RSyncPath) > 0 {
		rsyncExecutable = config.RSyncPath
	}
	excludesString := ""
	if len(excludes) > 0 {
		for _, exclude := range excludes {
			excludesString += fmt.Sprintf("%s ", exclude)
		}
	}
	return fmt.Sprintf("%s -avrhLK --delete --exclude \"%s\" %s/ %s", rsyncExecutable, excludesString, srcDir, dstDir)
}

func GetSnapshotDirName(snapshotName string, number int) string {
	return fmt.Sprintf("%s.%s", snapshotName, strconv.Itoa(number))
}

func executeOnlySnapshot(config *structs.Config, snapshotConfig *structs.SnapshotConfig) error {
	snapshotLogPrefix := fmt.Sprintf("[%s]", snapshotConfig.SnapshotName)
	before := time.Now().UnixMilli()
	newestSnapshotPath := path.Join(snapshotConfig.SnapshotsDir, GetSnapshotDirName(snapshotConfig.SnapshotName, 0))
	err := os.MkdirAll(snapshotConfig.SnapshotsDir, 0700)
	if err != nil {
		fmt.Println("hi")
		return fmt.Errorf("%s can't create snapshot dir %s: %s", snapshotLogPrefix, snapshotConfig.SnapshotsDir, err.Error())
	}
	tmpDir, mkdirErr := os.MkdirTemp(snapshotConfig.SnapshotsDir, "tmp")

	// in case of errors be sure to remove the tmp directory to avoid creating junk
	defer os.RemoveAll(tmpDir)
	if mkdirErr != nil {
		return fmt.Errorf("%s can't create tmp dir %s: %s", snapshotLogPrefix, tmpDir, mkdirErr.Error())
	}
	_, err = os.Stat(newestSnapshotPath)
	// if the snapshot 0 already exists, copy it with hard links into the tmp dir
	if err == nil {
		slog.Debug(fmt.Sprintf("%s copying latest snapshot...", snapshotLogPrefix))
		cpCommand := fmt.Sprintf("%s -lra %s/./ %s", config.CpPath, newestSnapshotPath, tmpDir)
		slog.Debug(fmt.Sprintf("%s running %s", snapshotLogPrefix, cpCommand))
		rsyncOutput, cpErr := exec.Command("sh", "-c", cpCommand).CombinedOutput()
		if cpErr != nil {
			return fmt.Errorf("%s error copying last snapshot %s to %s: %s, %s", snapshotLogPrefix, newestSnapshotPath, tmpDir, cpErr.Error(), string(rsyncOutput))
		}
	} else if os.IsNotExist(err) {
		slog.Debug(fmt.Sprintf("%s creating first snapshot %s", snapshotLogPrefix, newestSnapshotPath))
	} else {
		return fmt.Errorf("%s can't stat %s: %s", newestSnapshotPath, snapshotLogPrefix, err.Error())
	}
	now := time.Now()
	os.Chtimes(tmpDir, now, now)

	for _, dirToSnapshot := range snapshotConfig.Dirs {
		_, err = os.Stat(dirToSnapshot.SrcDirAbspath)
		if os.IsNotExist(err) {
			slog.Warn(fmt.Sprintf("%s source directory %s does not exist", snapshotLogPrefix, dirToSnapshot.SrcDirAbspath))
			continue
		}
		dstDirFull := path.Join(tmpDir, dirToSnapshot.DstDirInSnapshot)
		_, err = os.Stat(dstDirFull)
		if os.IsNotExist(err) {
			err = os.MkdirAll(dstDirFull, 0700)
			if err != nil {
				return fmt.Errorf("%s can't create destination dir %s", snapshotLogPrefix, dstDirFull)
			}
		}
		rsyncCommand := getRsyncDirsCommand(config, dirToSnapshot.SrcDirAbspath, dstDirFull, dirToSnapshot.Excludes)
		slog.Debug(fmt.Sprintf("%s running %s", snapshotLogPrefix, rsyncCommand))
		rsyncOutput, err := exec.Command("sh", "-c", rsyncCommand).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s can't sync %s/ to %s: %s, %s", snapshotLogPrefix, dirToSnapshot.SrcDirAbspath, dstDirFull, err.Error(), string(rsyncOutput))
		}
	}

	// rename all the snapshots
	snapshotsNumbers := []int{}
	snapshots, err := os.ReadDir(snapshotConfig.SnapshotsDir)
	if err != nil {
		return fmt.Errorf("%s can't read directory %s: %s", snapshotLogPrefix, snapshotConfig.SnapshotName, err.Error())
	}
	snapshotPrefixWithNumberRegex, err := regexp.Compile(fmt.Sprintf("^%s.([0-9]+)$", regexp.QuoteMeta(snapshotConfig.SnapshotName)))
	if err != nil {
		return fmt.Errorf("%s error compiling regex: %s", snapshotLogPrefix, err.Error())
	}
	for _, snapshot := range snapshots {
		match := snapshotPrefixWithNumberRegex.FindStringSubmatch(snapshot.Name())
		if match != nil {
			number, err := strconv.Atoi(match[1]) // match[1] contains the first capturing group
			if err != nil {
				return fmt.Errorf("%s error converting string to int: %s", snapshotLogPrefix, err.Error())
			}
			snapshotsNumbers = append(snapshotsNumbers, number)
		}
	}
	slices.Sort(snapshotsNumbers)
	slices.Reverse(snapshotsNumbers)
	for _, number := range snapshotsNumbers {
		snapshotOldName := GetSnapshotDirName(snapshotConfig.SnapshotName, number)
		snapshotOldPath := path.Join(snapshotConfig.SnapshotsDir, snapshotOldName)
		snapshotRenamedName := GetSnapshotDirName(snapshotConfig.SnapshotName, number+1)
		snapshotRenamedPath := path.Join(snapshotConfig.SnapshotsDir, snapshotRenamedName)
		slog.Debug(fmt.Sprintf("%s renaming %s to %s", snapshotLogPrefix, snapshotOldPath, snapshotRenamedName))
		err = os.Rename(snapshotOldPath, snapshotRenamedPath)
		if err != nil {
			return fmt.Errorf("%s can't move %s to %s: %s", snapshotLogPrefix, snapshotOldPath, snapshotOldPath, err.Error())
		}
	}

	// rename the temporary folder to be the newest snapshot
	slog.Debug(fmt.Sprintf("%s renaming tmp dir %s to %s", snapshotLogPrefix, tmpDir, newestSnapshotPath))
	err = os.Rename(tmpDir, newestSnapshotPath)
	if err != nil {
		return fmt.Errorf("%s can't rename temp directory %s to %s: %s", snapshotLogPrefix, tmpDir, newestSnapshotPath, err.Error())
	}

	// delete the excess amount of snapshots
	snapshots, err = os.ReadDir(snapshotConfig.SnapshotsDir)
	if err != nil {
		return fmt.Errorf("%s can't read directory %s: %s", snapshotLogPrefix, snapshotConfig.SnapshotsDir, err.Error())
	}
	for _, snapshotEntry := range snapshots {
		snapshotInfo, err := utils.GetInfoFromSnapshotPath(snapshotEntry.Name())
		if err != nil {
			return err
		}
		if snapshotInfo.Number >= snapshotConfig.Retention {
			snapshotToRemoveName := GetSnapshotDirName(snapshotConfig.SnapshotName, snapshotInfo.Number)
			snapshotToRemovePath := path.Join(snapshotConfig.SnapshotsDir, snapshotToRemoveName)
			err = os.RemoveAll(snapshotToRemovePath)
			if err != nil {
				return fmt.Errorf("%s can't remove snapshot %s: %s", snapshotLogPrefix, snapshotToRemovePath, err.Error())
			}
		}
	}

	after := time.Now().UnixMilli()
	seconds := float64(after-before) / 1000
	slog.Info(fmt.Sprintf("%s snapshots done in %.2f s", snapshotLogPrefix, seconds))
	return nil
}

func ExecuteSnapshot(config *structs.Config, snapshotConfig *structs.SnapshotConfig) error {
	snapshotLogPrefix := fmt.Sprintf("[%s]", snapshotConfig.SnapshotName)
	before := time.Now().UnixMilli()
	if len(snapshotConfig.PreSnapshotCommands) > 0 {
		slog.Info(fmt.Sprintf("%s executing pre snapshot commands", snapshotLogPrefix))
		for _, command := range snapshotConfig.PreSnapshotCommands {
			slog.Info(snapshotLogPrefix + " " + command)
			result, err := exec.Command("sh", "-c", command).Output()
			if err != nil {
				fmt.Println(snapshotLogPrefix + command + ": " + err.Error())
				return err
			}
			if len(result) > 0 {
				slog.Info(snapshotLogPrefix + command + ": " + string(result))
			}
		}
		after := time.Now().UnixMilli()
		seconds := float64(after-before) / 1000
		slog.Info(fmt.Sprintf("%s pre snapshot commands done in %.2f s", snapshotLogPrefix, seconds))
	} else {
		slog.Info(fmt.Sprintf("%s no pre snapshot commands to run", snapshotLogPrefix))
	}

	err := executeOnlySnapshot(config, snapshotConfig)
	if err != nil {
		if !snapshotConfig.AlwaysRunPostSnapshotCommands {
			return err
		}
		slog.Error(err.Error())
	}

	if len(snapshotConfig.PostSnapshotCommands) > 0 {
		slog.Info(fmt.Sprintf("%s executing post snapshot commands", snapshotLogPrefix))
		for _, command := range snapshotConfig.PostSnapshotCommands {
			slog.Info(fmt.Sprintf("%s %s", snapshotLogPrefix, command))
			result, err := exec.Command("sh", "-c", command).Output()
			if err != nil {
				return fmt.Errorf("%s %s: %s", snapshotLogPrefix, command, err.Error())
			}
			if len(result) > 0 {
				slog.Info(snapshotLogPrefix + command + ": " + string(result))
			}
		}
		after := time.Now().UnixMilli()
		seconds := float64(after-before) / 1000
		slog.Info(fmt.Sprintf("%s post snapshot commands done in %.2f s", snapshotLogPrefix, seconds))
	} else {
		slog.Info(fmt.Sprintf("%s no post snapshot commands to run", snapshotLogPrefix))
	}

	now := time.Now()
	os.Chtimes(GetSnapshotDirName(snapshotConfig.SnapshotName, 0), now, now)
	return nil
}

func GetSnapshotsInfo(snapshotsConfigsDir string, expandVars bool, snapshotName string) (snapshotsInfo []*structs.SnapshotInfo, err error) {
	snapshotConfig, err := configs.GetSnapshotConfigByName(snapshotsConfigsDir, expandVars, snapshotName)
	if err != nil {
		return snapshotsInfo, fmt.Errorf("can't list snapshots of %s: %s", snapshotName, err.Error())
	}
	if snapshotConfig == nil {
		slog.Warn("Snapshot template " + snapshotName + " does not exist.")
		return snapshotsInfo, nil
	}
	_, err = os.Stat(snapshotConfig.SnapshotsDir)
	if os.IsNotExist(err) {
		slog.Info("No snapshots found for " + snapshotName)
		return snapshotsInfo, nil
	}
	snapshotsDirsEntries, err := os.ReadDir(snapshotConfig.SnapshotsDir)
	if err != nil {
		return snapshotsInfo, fmt.Errorf("can't list snapshot of %s: %s", snapshotName, err.Error())
	}
	for _, entry := range snapshotsDirsEntries {
		snapshotFullPath := path.Join(snapshotConfig.SnapshotsDir, entry.Name())
		_, err := os.Stat(snapshotFullPath)
		if err != nil {
			return snapshotsInfo, fmt.Errorf("can't stat %s: %s", snapshotFullPath, err.Error())
		}
		snapshotInfo, err := utils.GetInfoFromSnapshotPath(path.Join(snapshotConfig.SnapshotsDir, entry.Name()))
		if err != nil {
			return snapshotsInfo, fmt.Errorf("can't parse snapshot name: %s", err.Error())
		}
		snapshotsInfo = append(snapshotsInfo, snapshotInfo)
	}
	return snapshotsInfo, nil
}

func RestoreSnapshot(config *structs.Config, number int, snapshotConfig *structs.SnapshotConfig) (err error) {
	snapshotLogPrefix := fmt.Sprintf("[%s]", snapshotConfig.SnapshotName)
	for _, dir := range snapshotConfig.Dirs {
		err = os.MkdirAll(dir.SrcDirAbspath, 0700)
		if err != nil {
			return fmt.Errorf("can't create directory %s: %s", dir.SrcDirAbspath, err.Error())
		}

		snapshottedDirPath := path.Join(snapshotConfig.SnapshotsDir, fmt.Sprintf("%s.%d", snapshotConfig.SnapshotName, number), dir.DstDirInSnapshot)
		rsyncCommand := getRsyncDirsCommand(config, snapshottedDirPath, dir.SrcDirAbspath, nil)
		slog.Debug(rsyncCommand)
		_, err = exec.Command("sh", "-c", rsyncCommand).Output()
		if err != nil {
			slog.Error(fmt.Sprintf("%s can't sync %s/ to %s: %s", snapshotLogPrefix, snapshottedDirPath, dir.SrcDirAbspath, err.Error()))
		}
	}
	return err
}
