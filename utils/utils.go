package utils

import (
	"fmt"
	"path"
	"snapsync/structs"
	"strconv"
	"strings"
)

func HumanReadableSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func GetInfoFromSnapshotBasePath(snapshotPath string) (snapshotInfo *structs.SnapshotInfo, err error) {
	items := strings.Split(snapshotPath, ".")
	if len(items) != 3 {
		return nil, fmt.Errorf("snapshot name must be in format <name>.<interval>.<number>")
	}
	name := items[0]
	interval := items[1]
	number, err := strconv.Atoi(items[2])
	if err != nil {
		return nil, fmt.Errorf("can't parse snapshot number: %s", snapshotPath)
	}
	return &structs.SnapshotInfo{
		Abspath:      "",
		SnapshotName: name,
		Interval:     interval,
		Number:       number,
	}, nil
}

func GetInfoFromSnapshotPath(snapshotPath string) (snapshotInfo *structs.SnapshotInfo, err error) {
	snapshotDirName := strings.TrimSuffix(path.Base(snapshotPath), "/")
	snapshotInfo, err = GetInfoFromSnapshotBasePath(snapshotDirName)
	if err != nil {
		return nil, err
	}
	snapshotInfo.Abspath = snapshotPath
	return snapshotInfo, nil
}
