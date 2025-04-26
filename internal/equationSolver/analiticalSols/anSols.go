package analiticalSols

import (
	"masters/internal/config"
	"math"
)

// решение для демфера и пружины
func GeneralAnalyticalSolution(t float64, conds *config.InitialConds) float64 {
	// Вычисление подкоренного выражения

	k := conds.K
	d := conds.D
	m := conds.M
	x0 := conds.X0
	v0 := conds.V0

	arg := d*d + 4*k*m

	sqrtArg := math.Sqrt(arg)

	// Вычисление экспоненциального множителя
	expFactor := math.Exp(-d * t / (2.0 * m))

	// Вычисление аргументов гиперболических функций
	hyperArg := (sqrtArg * t) / (2.0 * m)

	// Вычисление гиперболических функций
	cosh := math.Cosh(hyperArg)
	sinh := math.Sinh(hyperArg)
	// Первое слагаемое
	firstTerm := x0 * sqrtArg * cosh
	// Второе слагаемое
	secondTerm := (x0*d + 2.0*v0*m) * sinh
	// Сборка решения
	numerator := expFactor * (firstTerm + secondTerm)
	denominator := sqrtArg

	return expFactor * numerator / denominator
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
