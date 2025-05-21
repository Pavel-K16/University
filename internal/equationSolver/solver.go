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
	"strings"
)

var (
	log = logger.LoggerInit()
)

func Solver(conds *config.BodiesConds, timeConds *config.TimeConds) ([]float64, error) {

	var n int
	var err error
	var analit float64

	t0, t, tau := timeConds.T0, timeConds.T, timeConds.Tau

	body := &conds.Body1

	n = int(math.Round((t - t0) / tau))

	X, V, A := make([]float64, 1), make([]float64, 1), make([]float64, 1)

	X[0], V[0], A[0] = body.X0, body.V0, body.X0

	for i := 0; i < n; i++ {

		if err == nil {
			if body.D == 0 {
				analit, err = aS.SpringAnalyticalSolution(t0+tau*float64(i+1), body)
			} else {
				analit, err = aS.GeneralAnalyticalSolution(t0+tau*float64(i+1), body)
			}
		}

		A = append(A, analit)

		x, v := nM.RK4Method(X, V, i, timeConds, body)

		X = append(X, x)
		V = append(V, v)
	}

	//log.Debugf("len X: %v Num Sol: \n %v \n", len(X), X)

	if err = WriteNumSolutionToFile(X, body, timeConds, defaults.Points1FilePath); err != nil {
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

	//	log.Debugf("len A %v Analitic Sol: \n %v \n", len(A), A)

	return X, nil
}

func WriteNumSolutionToFile(X []float64, body *config.Body, timeConds *config.TimeConds, path string) error {

	pointsFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	t0, t, tau := timeConds.T0, timeConds.T, timeConds.Tau

	for i, val := range X {
		fmt.Fprintf(pointsFile, "%.10f ", t0+float64(i)*tau)
		fmt.Fprintf(pointsFile, "%.10f\n", val)
	}

	var paramsPath string

	splittedPath := strings.Split(path, "/")
	nameFile := splittedPath[len(splittedPath)-1]
	splittedNameFile := strings.Split(nameFile, ".")[0]

	if strings.HasSuffix(splittedNameFile, "1") {
		paramsPath = defaults.Params1FilePath
	} else if strings.HasSuffix(splittedNameFile, "2") {
		paramsPath = defaults.Params2FilePath
	}

	paramsFile, err := os.OpenFile(paramsPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	k, m, d, x0, v0 := body.K, body.M, body.D, body.X0, body.V0

	fmt.Fprintf(paramsFile, "%f \n%f \n%f \n%f \n%f \n%f \n%f", k, m, d, t0, t, x0, v0)

	return nil
}

func SolverCoupled(coupledConds *config.BodiesConds, timeConds *config.TimeConds) error {

	var n int
	var err error

	t0, t, tau := timeConds.T0, timeConds.T, timeConds.Tau

	n = int(math.Round((t - t0) / tau))

	body1 := &coupledConds.Body1
	body2 := &coupledConds.Body2

	X1, V1, X2, V2 := make([]float64, 1), make([]float64, 1), make([]float64, 1), make([]float64, 1)

	X1[0], V1[0], X2[0], V2[0] = body1.X0, body1.V0, body2.X0, body2.V0

	for i := 0; i < n; i++ {

		x1, v1, x2, v2 := nM.RK2MethodCoupled(tau, X1, V1, X2, V2, i, coupledConds)

		X1 = append(X1, x1)
		V1 = append(V1, v1)
		X2 = append(X2, x2)
		V2 = append(V2, v2)
	}

	//log.Debugf("len X1: %v Num Sol: \n %v \n", len(X1), X1)
	//log.Debugf("len X1: %v Num Sol: \n %v \n", len(X2), X2)

	if err = WriteNumSolutionToFile(X1, body1, timeConds, defaults.Points1FilePath); err != nil {
		log.Errorf("%s", err)
	}

	if err = WriteNumSolutionToFile(X2, body2, timeConds, defaults.Points2FilePath); err != nil {
		log.Errorf("%s", err)
	}

	if err != nil {
		log.Warningf("%s", err)

		return err
	}

	return nil
}
