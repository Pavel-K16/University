package logger

import (
	"fmt"
	"io"
	"masters/defaults"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)


var levelColors = map[logrus.Level]string{
	logrus.DebugLevel: "\033[32m", // Cyan
	logrus.InfoLevel:  "\033[36m", // Green
	logrus.WarnLevel:  "\033[31m", // Yellow
	logrus.ErrorLevel: "\033[31m", // Red
	logrus.FatalLevel: "\033[35m", // Magenta
	logrus.PanicLevel: "\033[31m", // Red
}

// CustomFormatter реализует интерфейс logrus.Formatter
type CustomFormatter struct {
	TimestampFormat string
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Получаем короткое имя файла
	file := ""
	line := ""
	if entry.HasCaller() {
		file = filepath.Base(entry.Caller.File)
		line = fmt.Sprintf("%d", entry.Caller.Line)
	}

	// Получаем цвет для уровня лога
	color := levelColors[entry.Level]
	reset := "\033[0m"

	// Формируем уровень лога в верхнем регистре
	level := strings.ToUpper(entry.Level.String())

	// Собираем строку в нужном формате
	timestamp := entry.Time.Format(f.TimestampFormat)
	msg := fmt.Sprintf("%s %s%s%s %s:%s %s\n",
		timestamp,
		color,
		level,
		reset,
		file,
		line,
		entry.Message,
	)

	return []byte(msg), nil
}

func LoggerInit() *logrus.Logger {
	logger := logrus.New()

	logger.SetFormatter(&CustomFormatter{
		TimestampFormat: "15:04:05",
	})

	logger.SetLevel(defaults.LogLevel)

	file, err := os.OpenFile(defaults.LogsFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err == nil {
		logger.SetOutput(io.MultiWriter(os.Stdout, file))
	} else {
		logger.SetOutput(os.Stdout)
	}

	logger.SetReportCaller(true)

	return logger
}
