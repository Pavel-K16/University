package config

import (
	"encoding/json"
	"masters/internal/defaults"
	logger "masters/internal/logger"
	"os"
)

var (
	log = logger.LoggerInit()
)

func CondsInit(conditions *InitialConds) error {
	var condsInfo []byte
	var err error

	if condsInfo, err = ReadFile(defaults.ConfigFilePath); err != nil {
		log.Errorf("Can't read the file: %s", err)

		return err
	}

	if err = json.Unmarshal(condsInfo, conditions); err != nil {
		return err
	}

	return nil
}

func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	return data, nil
}
