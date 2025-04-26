package equationsolver

import (
	"fmt"
	"masters/internal/config"
	aS "masters/internal/equationSolver/analiticalSols"
	"masters/internal/logger"
	nM "masters/internal/numMethods"
	"math"
)

var (
	log = logger.LoggerInit()
)

func Solver(conds *config.InitialConds) ([]float64, error) {

	var n int

	t0, t, tau := conds.T0, conds.T, conds.Tau

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

		x, v := nM.RK4Method(tau, X, V, i, conds)

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
