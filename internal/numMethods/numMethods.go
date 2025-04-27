package numMethods

import (
	"errors"
	"masters/internal/config"
	"masters/internal/logger"
	u "masters/internal/numMethods/utils"
)

var (
	log = logger.LoggerInit()
)

func RK2Method(tau float64, X, V []float64, i int, conds *config.InitialConds) (float64, float64) {
	log.Tracef("RK2Method.numMethods.go")
	t0 := conds.T0
	t := tau*float64(i) + t0
	vec0 := make([]float64, 2)
	vec0[0] = X[i]
	vec0[1] = V[i]
	k1, _ := F(t, vec0, conds)
	vec := u.VecAdd(vec0, u.VecMult(tau/2.0, k1))
	k2, _ := F(t+tau/2.0, vec, conds)

	x := X[i] + k2[0]*tau
	v := V[i] + k2[1]*tau

	return x, v
}

func RK4Method(tau float64, X, V []float64, i int, conds *config.InitialConds) (float64, float64) {
	log.Tracef("RK4Method.numMethods.go")

	t0 := conds.T0
	t := tau*float64(i) + t0
	vec0 := make([]float64, 2)

	vec0[0] = X[i]
	vec0[1] = V[i]

	k1, _ := F(t, vec0, conds)
	vec := u.VecAdd(vec0, u.VecMult(tau/2.0, k1))
	k2, _ := F(t+tau/2.0, vec, conds)

	vec = u.VecAdd(vec0, u.VecMult(tau/2.0, k2))
	k3, _ := F(t+tau/2.0, vec, conds)

	vec = u.VecAdd(vec0, u.VecMult(tau, k3))
	k4, _ := F(t+tau, vec, conds)

	K := u.VecAdd(k1, u.VecMult(2.0, k2), u.VecMult(2.0, k3), k4)
	K = u.VecMult(1.0/6.0, K)

	x := X[i] + K[0]*tau
	v := V[i] + K[1]*tau

	return x, v
}

func F(t float64, X []float64, conds *config.InitialConds) ([]float64, error) {
	log.Tracef("F.numMethods.go")

	if len(X) != 2 {
		err := errors.New("wrong size of vector X")
		log.Errorf("%s", err)

		return nil, err
	}

	k, m, d := u.InitConds4F(conds) // k m d

	vec := make([]float64, 2)

	vec[0] = X[1]
	vec[1] = (-k*X[0] - d*X[1]) / m

	return vec, nil
}
