package utils

import (
	"errors"
	"math"

	"masters/internal/logger"
)

var (
	log = logger.LoggerInit()
)

func Cnorm(a, b []float64) (float64, error) {

	if len(a) != len(b) {
		err := errors.New("different lengths of vectors")
		log.Errorf("len1: %v; len2: %v; %s", len(a), len(b), err)

		return 0, err
	}
	var max float64 = 0

	for i := range a {
		val := math.Abs(a[i] - b[i])
		if val > max {
			max = val
		}
	}

	return max, nil
}
