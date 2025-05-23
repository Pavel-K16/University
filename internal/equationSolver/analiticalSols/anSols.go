package analiticalSols

import (
	"errors"
	"masters/internal/config"
	"masters/internal/logger"
	u "masters/internal/numMethods/utils"
	"math"
)

var (
	log = logger.LoggerInit()
)

// решение для демфера и пружины
func GeneralAnalyticalSolution(t float64, conds *config.InitialConds) (float64, error) {
	// Вычисление подкоренного выражения

	k, m, d, x0, v0 := u.InitConds4aS(conds)

	arg := (d * d) - (4.0 * k * m)

	if arg <= 0 {
		err := errors.New("incorrect params 4 aS: d*d - 4*k*m = ")
		log.Warningf("%s %f", err, arg)

		return 0, err
	}

	sqrtArg := math.Sqrt(arg)

	// Вычисление экспоненциального множителя
	expFactor := math.Exp((-d * t) / (2.0 * m))
	// Вычисление аргументов гиперболических функций
	hyperArg := (sqrtArg * t) / (2.0 * m)
	// Вычисление гиперболических функций
	cosh := math.Cosh(hyperArg)
	sinh := math.Sinh(hyperArg)
	// Первое слагаемое
	firstTerm := x0 * cosh
	// Второе слагаемое
	secondTerm := (((x0 * d) + (2.0 * v0 * m)) * sinh) / sqrtArg
	// Сборка решения

	return expFactor * (firstTerm + secondTerm), nil
}

// Решение только для пружинки
func SpringAnalyticalSolution(t float64, conds *config.InitialConds) (float64, error) {

	k := conds.K
	m := conds.M
	x0 := conds.X0
	v0 := conds.V0

	koef := k / m
	omega := math.Sqrt(koef)

	firstTerm := x0 * math.Cos(omega*t)
	secondTerm := v0 * math.Sin(omega*t)

	return firstTerm + secondTerm, nil
}
