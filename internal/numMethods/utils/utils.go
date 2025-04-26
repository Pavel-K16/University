package utils

func VecAdd(a ...[]float64) []float64 {
	sumVec := make([]float64, 2)

	for _, val := range a {
		sumVec[0] += val[0]
		sumVec[1] += val[1]
	}

	return sumVec
}

func VecMult(k float64, a []float64) []float64 {
	vec := make([]float64, 2)

	vec[0] = a[0] * k
	vec[1] = a[1] * k

	return vec
}
