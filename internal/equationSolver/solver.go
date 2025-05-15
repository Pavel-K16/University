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

	if err = WriteNumSolutionToFile(X, conds, defaults.PointsFilePath); err != nil {
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

func WriteNumSolutionToFile(X []float64, conds *config.InitialConds, path string) error {

	pointsFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
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

func SolverCoupled(coupledConds *config.InitialCondsCoupled, conds *config.InitialConds) error {

	var n int
	var err error

	t0, t, tau := conds.T0, conds.T, conds.Tau

	n = int(math.Round((t - t0) / tau))

	X1, V1, X2, V2 := make([]float64, 1), make([]float64, 1), make([]float64, 1), make([]float64, 1)

	X1[0], V1[0], X2[0], V2[0] = coupledConds.X1, coupledConds.V1, coupledConds.X2, coupledConds.V2

	for i := 0; i < n; i++ {

		x1, v1, x2, v2 := nM.RK2MethodCoupled(tau, X1, V1, X2, V2, i, coupledConds)

		X1 = append(X1, x1)
		V1 = append(V1, v1)
		X2 = append(X2, x2)
		V2 = append(V2, v2)
	}

	log.Debugf("len X1: %v Num Sol: \n %v \n", len(X1), X1)

	if err = WriteNumSolutionToFile(X1, conds, defaults.PointsFilePath); err != nil {
		log.Errorf("%s", err)
	}

	if err = WriteNumSolutionToFile(X2, conds, defaults.Points2FilePath); err != nil {
		log.Errorf("%s", err)
	}

	if err != nil {
		log.Warningf("%s", err)

		return err
	}

	return nil
}
