package equationsolver

import (
	"fmt"
	"masters/config"
	aS "masters/equationSolver/analiticalSols"
	"masters/logger"
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

	k := conds.K
	d := conds.D
	m := conds.M

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
		if i > 0 {
			analit := aS.SpringAnalyticalSolution(float64(i)*tau, conds)
			A = append(A, analit)
		}

		//k = k * (-1)
		//d = d * (-1)

		k1x := V[i]
		k1v := -(k*X[i] + d*V[i]) / m

		// Вычисляем значения на половинном шаге
		xHalf := X[i] + (tau/2)*k1x
		vHalf := V[i] + (tau/2)*k1v

		// Вычисляем вторые коэффициенты
		k2x := vHalf
		k2v := -(k*xHalf + d*vHalf) / m

		// Финальное обновление
		x := X[i] + tau*k2x
		v := V[i] + tau*k2v

		//x := X[i] + tau*(V[i]+(tau/2.0)*V[i])
		//v := V[i] + (k*tau/m)*(V[i]+(tau/2)*(k*X[i]-d*V[i])/m) - (d*tau/m)*(V[i]+(tau/2)*(k*V[i]-d*V[i])/m)

		//x := X[i] + V[i]*tau
		//v := V[i] - tau*((k*X[i]-d*V[i])/m)

		X = append(X, x)
		V = append(V, v)
	}

	log.Debugf("len X: %v", len(X))
	log.Debugf("%v \n", X)
	log.Debugf("len V: %v", len(V))
	log.Debugf("%v", V)

	log.Debugf("len A: %v", len(A))
	log.Debugf("%v", A)

	return X, nil
}
