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

func CondsInit(bodiesConds *BodiesConds, timeConds *TimeConds) error {
	var bodiesInfo, timeInfo []byte

	var err error

	if bodiesInfo, err = ReadFile(defaults.BodiesConfigFilePath); err != nil {
		log.Errorf("Can't read the file: %s", err)

		return err
	}

	if timeInfo, err = ReadFile(defaults.TimeConfigPath); err != nil {
		log.Errorf("Can't read the file: %s", err)

		return err
	}

	if err = json.Unmarshal(bodiesInfo, bodiesConds); err != nil {
		log.Errorf("Unmarshal err: %s", err)

		return err
	}

	if err = json.Unmarshal(timeInfo, timeConds); err != nil {
		log.Errorf("Unmarshal err: %s", err)

		return err
	}

	if err = checkConfig(bodiesConds, timeConds); err != nil {
		log.Errorf("Wrong config data: %s", err)

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

func checkConfig(bodiesConds *BodiesConds, timeConds *TimeConds) error {
	tau, t, t0 := timeConds.Tau, timeConds.T, timeConds.T0
	body1 := bodiesConds.Body1
	body2 := bodiesConds.Body2
	connParams := bodiesConds.ConnParams

	m1, k1, d1, m2, k2, d2 := body1.M, body1.K, body1.D, body2.M, body2.K, body2.D
	k12, d12 := connParams.K12, connParams.D12

	if tau < 0 || t < t0 || d1 < 0 || k1 < 0 || m1 < 0 || d2 < 0 || k2 < 0 || m2 < 0 || d12 < 0 || k12 < 0 {
		err := errors.New("incorrect json conds input")
		log.Errorf("%s", err)

		return nil
	}

	return nil
}
