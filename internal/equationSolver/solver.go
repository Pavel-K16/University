package equationsolver

import (
	"fmt"
	"masters/internal/config"
	"masters/internal/defaults"
	aS "masters/internal/equationSolver/analiticalSols"
	"masters/internal/logger"
	nM "masters/internal/numMethods"
	u "masters/internal/numMethods/utils"
	"math"
	"os"
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
			analit, err = aS.GeneralAnalyticalSolution(t0+tau*float64(i+1), conds)
			A = append(A, analit)
		}

		x, v := nM.RK4Method(tau, X, V, i, conds)

		X = append(X, x)
		V = append(V, v)
	}

	log.Debugf("len X: %v Num Sol: \n %v \n", len(X), X)

	if err = WriteNumSolutionToFile(X, conds); err != nil {
		log.Errorf("%s", err)
	}

	if err != nil {
		log.Warningf("%s", err)

		return X, err
	}

	if diff, err := u.Cnorm(A, X); err == nil {
		log.Debugf("Cnorm: %.20f", diff)
	} else {
		log.Errorf("%s", err)

		return X, nil
	}

	log.Debugf("len A %v Analitic Sol: \n %v \n", len(A), A)

	return X, nil
}

func WriteNumSolutionToFile(X []float64, conds *config.InitialConds) error {

	pointsFile, err := os.OpenFile(defaults.PointsFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	tau, t0 := conds.Tau, conds.T0

	for i, val := range X {
		fmt.Fprintf(pointsFile, "%.10f ", t0+float64(i)*tau)
		fmt.Fprintf(pointsFile, "%.10f\n", val)
	}

	paramsFile, err := os.OpenFile(defaults.ParamsFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	k, m, d, t, x0, v0 := conds.K, conds.M, conds.D, conds.T, conds.X0, conds.V0
	fmt.Fprintf(paramsFile, "%f \n%f \n%f \n%f \n%f \n%f \n%f", k, m, d, t0, t, x0, v0)

	return nil
}
