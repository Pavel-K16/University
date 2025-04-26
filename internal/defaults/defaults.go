package defaults

import "github.com/sirupsen/logrus"

const (
	ConfigFilePath = "../internal/config/config.json"
	LogsFilePath   = "../internal/logger/logs/logs.txt"
	LogLevel       = logrus.TraceLevel
)
