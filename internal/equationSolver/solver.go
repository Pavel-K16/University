package equationsolver

import (
	"errors"
	"fmt"
	"masters/internal/config"
	aS "masters/internal/equationSolver/analiticalSols"
	"masters/internal/logger"
	"math"
)

var (
	log = logger.LoggerInit()
)

func Solver(conds *config.InitialConds) ([]float64, error) {

	var n int

	t0 := conds.T0
	t := conds.T
	tau := conds.Tau

	//if tau < 0 || t < t0 || k < 0 || d < 0 || m < 0 {
	//	err := errors.New("incorrect json conds input")
	//	log.Errorf("%s", err)

	//	return nil, err
	//}

	n = int(math.Round((t - t0) / tau))

	fmt.Println("num of points:", n)

	X := make([]float64, 1)
	V := make([]float64, 1)
	A := make([]float64, 1)

	X[0] = conds.X0
	V[0] = conds.V0
	A[0] = conds.X0

	for i := 0; i < n; i++ {
		//if i > 0 {

		//log.Debugf("t: %f", tau*float64(i+1))

		analit := aS.GeneralAnalyticalSolution(tau*float64(i+1), conds)
		A = append(A, analit)

		x, v := RK4Method(tau, X, V, i, conds)

		X = append(X, x)
		V = append(V, v)
	}

	log.Debugf("len X Численное решение: %v", len(X))
	log.Debugf("%v \n", X)
	//log.Debugf("len V: %v", len(V))
	//log.Debugf("%v", V)

	log.Debugf("len A Аналитическое решение: %v", len(A))
	log.Debugf("%v", A)

	return X, nil
}

func RK2Method(tau float64, X, V []float64, i int, conds *config.InitialConds) (float64, float64) {

	t := tau * float64(i)
	vec0 := make([]float64, 2)
	vec0[0] = X[i]
	vec0[1] = V[i]
	k1, _ := F(t, vec0, conds)
	vec := VecAdd(vec0, VecMult(tau/2.0, k1))
	k2, _ := F(t+tau/2.0, vec, conds)

	x := X[i] + k2[0]*tau
	v := V[i] + k2[1]*tau

	return x, v
}

func RK4Method(tau float64, X, V []float64, i int, conds *config.InitialConds) (float64, float64) {

	t := tau * float64(i)
	vec0 := make([]float64, 2)
	vec0[0] = X[i]
	vec0[1] = V[i]
	k1, _ := F(t, vec0, conds)
	vec := VecAdd(vec0, VecMult(tau/2.0, k1))
	k2, _ := F(t+tau/2.0, vec, conds)

	vec = VecAdd(vec0, VecMult(tau/2.0, k2))
	k3, _ := F(t+tau/2.0, vec, conds)

	vec = VecAdd(vec0, VecMult(tau, k3))
	k4, _ := F(t+tau, vec, conds)

	K := VecAdd(k1, VecMult(2.0, k2), VecMult(2.0, k3), k4)
	K = VecMult(1.0/6.0, K)

	x := X[i] + K[0]*tau
	v := V[i] + K[1]*tau

	return x, v
}

func F(t float64, X []float64, conds *config.InitialConds) ([]float64, error) {

	if len(X) != 2 {
		err := errors.New("wrong size vector X")
		log.Errorf("%s", err)
		return nil, err
	}

	k := conds.K
	m := conds.M
	d := conds.D

	vec := make([]float64, 2)

	vec[0] = X[1]
	vec[1] = (-k*X[0] - d*X[1]) / m

	return vec, nil
}

func VecAdd(a ...[]float64) []float64 {

	sumVec := make([]float64, 2)

	for _, val := range a {
		sumVec[0] += val[0]
		sumVec[1] += val[1]
	}

	return sumVec
}

func VecMult(k float64, a []float64) []float64 {
	vec := make([]float64, 2)

	vec[0] = a[0] * k
	vec[1] = a[1] * k

	return vec
}
