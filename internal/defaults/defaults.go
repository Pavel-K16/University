package defaults

import "github.com/sirupsen/logrus"

const (
	ConfigFilePath = "../internal/config/config.json"
	LogsFilePath   = "../internal/logger/logs/logs.txt"
	LogLevel       = logrus.DebugLevel
	PointsFilePath = "../wolfram/paramsAndPoints/points.txt"
	ParamsFilePath = "../wolfram/paramsAndPoints/params.txt"
)
