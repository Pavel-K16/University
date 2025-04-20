package equationsolver

import (
	"fmt"
	"masters/config"
	"math"
)

func Solver(conds *config.InitialConds) []float64 {

	var n int

	t0 := conds.T0
	t := conds.T
	tau := conds.Tau

	k := conds.K
	d := conds.D
	m := conds.M

	n = int(math.Round((t - t0) / tau))

	fmt.Println("num of points:", n)

	X := make([]float64, 1)
	V := make([]float64, 1)

	X[0] = conds.X0
	V[0] = conds.V0
	for i := 0; i < n; i++ {
		x := X[i] + tau*(V[i]+(tau/2.0)*V[i])
		v := V[i] + (k*tau/m)*(V[i]+(tau/2)*(k*X[i]-d*V[i])/m)

		X = append(X, x)
		V = append(V, v)
	}
	
	fmt.Println("len X:", len(X))
	fmt.Printf("%v \n", X)
	fmt.Println("len V:", len(V))
	fmt.Printf("%v", V)

	return nil
}
