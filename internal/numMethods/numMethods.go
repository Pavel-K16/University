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

func RK2Method(X, V []float64, i int, timeConds *config.TimeConds, conds *config.Body) (float64, float64) {
	log.Tracef("RK2Method.numMethods.go")

	t0 := timeConds.T0
	tau := timeConds.Tau
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

func RK4Method(X, V []float64, i int, timeConds *config.TimeConds, conds *config.Body) (float64, float64) {
	log.Tracef("RK4Method.numMethods.go")

	t0 := timeConds.T0
	tau := timeConds.Tau
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

func F(t float64, X []float64, body *config.Body) ([]float64, error) {
	log.Tracef("F.numMethods.go")

	if len(X) != 2 {
		err := errors.New("wrong size of vector X")
		log.Errorf("%s", err)

		return nil, err
	}

	k, m, d := body.K, body.M, body.D

	vec := make([]float64, 2)

	vec[0] = X[1]
	vec[1] = (-k*X[0] - d*X[1]) / m

	return vec, nil
}

// RK2MethodCoupled решает систему связанных колебаний двух масс методом Рунге-Кутты второго порядка
// Система:
// m₁(d²x₁/dt²) + d₁x₁' + k₁x₁ - F₁₂ = 0
// m₂(d²x₂/dt²) + d₂x₂' + k₂x₂ + F₁₂ = 0
// F₁₂ + k₁₂(x₁ - x₂) + d₁₂(x₁ - x₂) = 0
func RK2MethodCoupled(tau float64, X1, V1, X2, V2 []float64, i int, conds *config.BodiesConds) (float64, float64, float64, float64) {
	// params: m1, m2, k1, k2, d1, d2, k12, d12
	body1 := conds.Body1
	body2 := conds.Body2

	m1 := body1.M
	m2 := body2.M
	k1 := body1.K
	k2 := body2.K
	d1 := body1.D
	d2 := body2.D
	k12 := conds.ConnParams.K12
	d12 := conds.ConnParams.D12

	x1 := X1[i]
	v1 := V1[i]
	x2 := X2[i]
	v2 := V2[i]

	// Сила связи на предыдущем слое
	// F12 := -k12*(x1-x2) - d12*(v1-v2) // не используется

	// Правая часть для x1, v1
	f1 := func(x1, v1, x2, v2 float64) (dx1, dv1 float64) {
		F12 := -k12*(x1-x2) - d12*(v1-v2)
		dx1 = v1
		dv1 = (-k1*x1 - d1*v1 + F12) / m1
		return
	}
	// Правая часть для x2, v2
	f2 := func(x1, v1, x2, v2 float64) (dx2, dv2 float64) {
		F12 := -k12*(x1-x2) - d12*(v1-v2)
		dx2 = v2
		dv2 = (-k2*x2 - d2*v2 - F12) / m2
		return
	}

	// k1 для обеих масс
	dx1_1, dv1_1 := f1(x1, v1, x2, v2)
	dx2_1, dv2_1 := f2(x1, v1, x2, v2)

	// k2 для обеих масс (на полушаге)
	dx1_2, dv1_2 := f1(x1+tau/2*dx1_1, v1+tau/2*dv1_1, x2+tau/2*dx2_1, v2+tau/2*dv2_1)
	dx2_2, dv2_2 := f2(x1+tau/2*dx1_1, v1+tau/2*dv1_1, x2+tau/2*dx2_1, v2+tau/2*dv2_1)

	nx1 := x1 + tau*dx1_2
	nv1 := v1 + tau*dv1_2
	nx2 := x2 + tau*dx2_2
	nv2 := v2 + tau*dv2_2

	return nx1, nv1, nx2, nv2
}

// RK4MethodCoupled решает систему связанных колебаний двух масс методом Рунге-Кутты 4-го порядка
// Система:
// m₁(d²x₁/dt²) + d₁x₁' + k₁x₁ - F₁₂ = 0
// m₂(d²x₂/dt²) + d₂x₂' + k₂x₂ + F₁₂ = 0
// F₁₂ + k₁₂(x₁ - x₂) + d₁₂(x₁' - x₂') = 0
func RK4MethodCoupled(tau float64, X1, V1, X2, V2 []float64, i int, params map[string]float64) (float64, float64, float64, float64) {
	m1 := params["m1"]
	m2 := params["m2"]
	k1 := params["k1"]
	k2 := params["k2"]
	d1 := params["d1"]
	d2 := params["d2"]
	k12 := params["k12"]
	d12 := params["d12"]

	x1 := X1[i]
	v1 := V1[i]
	x2 := X2[i]
	v2 := V2[i]

	f1 := func(x1, v1, x2, v2 float64) (dx1, dv1 float64) {
		F12 := -k12*(x1-x2) - d12*(v1-v2)
		dx1 = v1
		dv1 = (-k1*x1 - d1*v1 + F12) / m1
		return
	}
	f2 := func(x1, v1, x2, v2 float64) (dx2, dv2 float64) {
		F12 := -k12*(x1-x2) - d12*(v1-v2)
		dx2 = v2
		dv2 = (-k2*x2 - d2*v2 - F12) / m2
		return
	}

	// k1
	dx1_1, dv1_1 := f1(x1, v1, x2, v2)
	dx2_1, dv2_1 := f2(x1, v1, x2, v2)

	// k2
	dx1_2, dv1_2 := f1(x1+tau/2*dx1_1, v1+tau/2*dv1_1, x2+tau/2*dx2_1, v2+tau/2*dv2_1)
	dx2_2, dv2_2 := f2(x1+tau/2*dx1_1, v1+tau/2*dv1_1, x2+tau/2*dx2_1, v2+tau/2*dv2_1)

	// k3
	dx1_3, dv1_3 := f1(x1+tau/2*dx1_2, v1+tau/2*dv1_2, x2+tau/2*dx2_2, v2+tau/2*dv2_2)
	dx2_3, dv2_3 := f2(x1+tau/2*dx1_2, v1+tau/2*dv1_2, x2+tau/2*dx2_2, v2+tau/2*dv2_2)

	// k4
	dx1_4, dv1_4 := f1(x1+tau*dx1_3, v1+tau*dv1_3, x2+tau*dx2_3, v2+tau*dv2_3)
	dx2_4, dv2_4 := f2(x1+tau*dx1_3, v1+tau*dv1_3, x2+tau*dx2_3, v2+tau*dv2_3)

	nx1 := x1 + tau/6*(dx1_1+2*dx1_2+2*dx1_3+dx1_4)
	nv1 := v1 + tau/6*(dv1_1+2*dv1_2+2*dv1_3+dv1_4)
	nx2 := x2 + tau/6*(dx2_1+2*dx2_2+2*dx2_3+dx2_4)
	nv2 := v2 + tau/6*(dv2_1+2*dv2_2+2*dv2_3+dv2_4)

	return nx1, nv1, nx2, nv2
}
