package configs

import (
	"fmt"
	"os"
	"path"
	"snapsync/structs"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadConfig(configsDir string, expandVars bool) (config *structs.Config, err error) {
	configPath := path.Join(configsDir, "config.yml")
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("can't read %s: %s", configPath, err.Error())
	}
	configFileContent := string(configFile)
	if expandVars {
		configFileContent = os.ExpandEnv(configFileContent)
	}
	config = &structs.Config{}
	err = yaml.Unmarshal([]byte(configFileContent), config)
	if err != nil {
		return nil, fmt.Errorf("can't parse %s: %s", configPath, err.Error())
	}
	return config, nil
}

func LoadSnapshotsConfigs(snapshotsConfigsDir string, expandVars bool) (snapshotsConfigs []*structs.SnapshotConfig, err error) {
	snapshotConfigsEntries, err := os.ReadDir(snapshotsConfigsDir)
	if err != nil {
		return snapshotsConfigs, fmt.Errorf("can't read directory %s: %s", snapshotsConfigsDir, err.Error())
	}
	for _, snapshotConfigEntry := range snapshotConfigsEntries {
		if strings.HasPrefix(snapshotConfigEntry.Name(), "config.yml") {
			continue
		}
		if !strings.HasSuffix(snapshotConfigEntry.Name(), ".yml") {
			continue
		}
		absPath := path.Join(snapshotsConfigsDir, snapshotConfigEntry.Name())
		snapshotConfigFile, err := os.ReadFile(absPath)
		if err != nil {
			return snapshotsConfigs, fmt.Errorf("can't read snapshot config file %s: %s", absPath, err.Error())
		}
		configFileContent := string(snapshotConfigFile)
		if expandVars {
			configFileContent = os.ExpandEnv(configFileContent)
		}
		snapshotConfig := structs.SnapshotConfig{}
		err = yaml.Unmarshal([]byte(configFileContent), &snapshotConfig)
		if err != nil {
			return snapshotsConfigs, fmt.Errorf("can't parse snapshot config file %s: %s", absPath, err.Error())
		}
		for _, dir := range snapshotConfig.Dirs {
			if !path.IsAbs(dir.SrcDirAbspath) {
				return nil, fmt.Errorf("%s: src_dir_abspath must be an absolute path", snapshotConfig.SnapshotName)
			}
		}
		snapshotsConfigs = append(snapshotsConfigs, &snapshotConfig)
	}
	for i, snapshotConfig := range snapshotsConfigs {
		for j, sc := range snapshotsConfigs {
			if i == j {
				continue
			}
			if snapshotConfig.SnapshotName == sc.SnapshotName {
				return nil, fmt.Errorf("there are two snapshot configs with the same snapshot name: %s", sc.SnapshotName)
			}
		}
	}
	return snapshotsConfigs, nil
}

func ValidateSnapshotConfig(snapshotConfig *structs.SnapshotConfig) error {
	if strings.Contains(snapshotConfig.SnapshotName, ".") || strings.Contains(snapshotConfig.SnapshotName, " ") {
		return fmt.Errorf("snapshot %s'sname must not include dots or whitespaces", snapshotConfig.SnapshotName)
	}
	return nil
}

func GetDefaultConfigsDir() string {
	result, _ := os.Getwd()
	return result
}

func GetSnapshotConfigByName(snapshotsConfigsDir string, expandVars bool, snapshotName string) (*structs.SnapshotConfig, error) {
	snapshotConfigs, err := LoadSnapshotsConfigs(snapshotsConfigsDir, expandVars)
	if err != nil {
		return nil, err
	}
	for _, snapshotConfig := range snapshotConfigs {
		if snapshotConfig.SnapshotName == snapshotName {
			return snapshotConfig, nil
		}
	}
	return nil, nil
}
