package equationsolver

import (
	"masters/internal/config"
	aS "masters/internal/equationSolver/analiticalSols"
	"masters/internal/logger"
	nM "masters/internal/numMethods"
	u "masters/internal/numMethods/utils"
	"math"
)

var (
	log = logger.LoggerInit()
)

func Solver(conds *config.InitialConds) ([]float64, error) {

	var n int
	var err error
	var analit float64

	t0, t, tau := conds.T0, conds.T, conds.Tau

	n = int(math.Round((t - t0) / tau))

	X, V, A := make([]float64, 1), make([]float64, 1), make([]float64, 1)

	X[0], V[0], A[0] = conds.X0, conds.V0, conds.X0

	for i := 0; i < n; i++ {

		if err == nil {
			analit, err = aS.GeneralAnalyticalSolution(tau*float64(i+1), conds)
			A = append(A, analit)
		}

		x, v := nM.RK4Method(tau, X, V, i, conds)

		X = append(X, x)
		V = append(V, v)
	}

	log.Debugf("len X: %v Num Sol: \n %v \n", len(X), X)

	if err != nil {
		log.Warningf("%s", err)

		return X, nil
	}

	log.Debugf("len A %v Analitic Sol: \n %v \n", len(A), A)

	if diff, err := u.Cnorm(A, X); err == nil {
		log.Debugf("Cnorm: %.20f", diff)
	} else {
		log.Errorf("%s", err)
	}

	return X, nil
}
