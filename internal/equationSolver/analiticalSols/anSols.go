package analiticalSols

import (
	"masters/internal/config"
	"masters/internal/logger"
	u "masters/internal/numMethods/utils"
	"math"
)

var (
	log = logger.LoggerInit()
)

// решение для демфера и пружины
func GeneralAnalyticalSolution(t float64, conds *config.InitialConds) float64 {
	// Вычисление подкоренного выражения

	k, m, d, x0, v0 := u.InitConds4aS(conds)

	arg := (d * d) - (4.0 * k * m)

	sqrtArg := math.Sqrt(arg)
	log.Debugf("ARG: %f", sqrtArg)
	// Вычисление экспоненциального множителя
	expFactor := math.Exp((-d * t) / (2.0 * m))
	log.Debugf("EXP: %f", expFactor)
	// Вычисление аргументов гиперболических функций
	hyperArg := (sqrtArg * t) / (2.0 * m)

	// Вычисление гиперболических функций
	cosh := math.Cosh(hyperArg)
	log.Debugf("cosh: %f", cosh)
	sinh := math.Sinh(hyperArg)
	log.Debugf("sinh: %f", sinh)
	// Первое слагаемое
	firstTerm := x0 * cosh
	// Второе слагаемое
	secondTerm := (((x0 * d) + (2.0 * v0 * m)) * sinh) / sqrtArg
	// Сборка решения

	return expFactor * (firstTerm + secondTerm)
}

// Решение только для пружинки
func SpringAnalyticalSolution(t float64, conds *config.InitialConds) float64 {

	k := conds.K
	m := conds.M
	x0 := conds.X0
	v0 := conds.V0

	koef := k / m
	omega := math.Sqrt(koef)

	firstTerm := x0 * math.Cos(omega*t)
	secondTerm := v0 * math.Sin(omega*t)

	return firstTerm + secondTerm
}
