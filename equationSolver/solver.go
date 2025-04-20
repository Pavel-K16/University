package equationsolver

import (
	"errors"
	"fmt"
	"masters/config"
	"masters/logger"
	"math"
)

func Solver(conds *config.InitialConds) ([]float64, error) {

	var n int

	t0 := conds.T0
	t := conds.T
	tau := conds.Tau

	k := conds.K
	d := conds.D
	m := conds.M

	if tau < 0 || t < t0 || k < 0 || d < 0 || m < 0 {
		err := errors.New("incorrect json conds input")
		logger.Logger.Errorf("%s", err)

		return nil, err
	}

	n = int(math.Round((t - t0) / tau))

	fmt.Println("num of points:", n)

	X := make([]float64, 1)
	V := make([]float64, 1)

	X[0] = conds.X0
	V[0] = conds.V0

	for i := 0; i < n; i++ {
		//x := X[i] + tau*(V[i]+(tau/2.0)*V[i])
		//v := V[i] + (k*tau/m)*(V[i]+(tau/2)*(k*X[i]-d*V[i])/m) - (d*tau/m)*(V[i]+(tau/2)*(k*V[i]-d*V[i])/m)

		x := X[i] + V[i]*tau
		v := V[i] - tau*((k*X[i]-d*V[i])/m)

		X = append(X, x)
		V = append(V, v)
	}

	logger.Logger.Debugf("len X: %v", len(X))
	logger.Logger.Debugf("%v \n", X)
	logger.Logger.Debugf("len V: %v", len(V))
	logger.Logger.Debugf("%v", V)

	return X, nil
}
