package config

import (
	"encoding/json"
	"masters/defaults"
	logger "masters/logger"
	"os"
)

func CondsInit(conditions *InitialConds) error {
	var condsInfo []byte
	var err error

	if condsInfo, err = ReadFile(defaults.ConfigFilePath); err != nil { // TODO: Replace with const
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
		logger.Logger.Warningf("Ошибка при чтении файла: %s", err)

		return nil, err
	}

	return data, nil
}
