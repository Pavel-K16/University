package config

import (
	"encoding/json"
	"errors"

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
		log.Errorf("Unmarshal err: %s", err)

		return err
	}

	if err = checkConfig(conditions); err != nil {
		log.Errorf("Wrong config data: %s", err)

		return err
	}

	return nil
}

func CoupledCondsInit(conditions *InitialCondsCoupled) error {
	var condsInfo []byte
	var err error

	if condsInfo, err = ReadFile(defaults.ConfigCoupledFilePath); err != nil {
		log.Errorf("Can't read the file: %s", err)

		return err
	}

	if err = json.Unmarshal(condsInfo, conditions); err != nil {
		log.Errorf("Unmarshal err: %s", err)

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

func checkConfig(conditions *InitialConds) error {
	tau, t, t0, k, d, m := condsInit(conditions)

	if tau < 0 || t < t0 || d < 0 || k < 0 || m < 0 {
		err := errors.New("incorrect json conds input")
		log.Errorf("%s", err)

		return err
	}

	return nil
}

func condsInit(conditions *InitialConds) (float64, float64, float64, float64, float64, float64) {
	tau := conditions.Tau
	t := conditions.T
	t0 := conditions.T0
	k := conditions.K
	d := conditions.D
	m := conditions.M

	return tau, t, t0, k, d, m
}
