package aero

import "masters/internal/logger"

type influenceKoefMatrix struct {
	matrix [][]float64
	v      float64 // скорость
	p      float64 // плотность
	b      float64 // хорда профиля лопатки
	m      float64 // обобщённая масса лопатки
}

var (
	matrix   influenceKoefMatrix
	nodesNum []int
	log      = logger.LoggerInit()
)

func InitInfluenceKoefMatrix(numS []int) {
	nodesNum = numS

	matrix.matrix = make([][]float64, len(nodesNum))
	for i := range nodesNum {
		matrix.matrix[i] = make([]float64, len(nodesNum))
	}

	for i := range nodesNum {
		for j := range nodesNum {
			matrix.matrix[i][j] = 0
		}
	}
}

func GetInfluenceKoef(num1, num2 int) float64 {
	if num1 >= len(matrix.matrix) || num2 >= len(matrix.matrix[0]) {
		log.Errorf("num1 or num2 is out of range: %d, %d", num1, num2)

		return 0
	}

	mapNum := make(map[int]int)
	for i, num := range nodesNum {
		mapNum[num] = i
	}

	return matrix.matrix[mapNum[num1]][mapNum[num2]]
}

func SetInfluenceKoef(num1, num2 int, value float64) {
	if num1 >= len(matrix.matrix) || num2 >= len(matrix.matrix[0]) {
		log.Errorf("num1 or num2 is out of range: %d, %d", num1, num2)

		return
	}

	mapNum := make(map[int]int)
	for i, num := range nodesNum {
		mapNum[num] = i
	}

	matrix.matrix[mapNum[num1]][mapNum[num2]] = value
	matrix.matrix[mapNum[num2]][mapNum[num1]] = value
}

func SetFlowParameters(v, p, b, m float64) { // скорость, плотность, хорда профиля лопатки, обобщённая масса лопатки
	matrix.v = v
	matrix.p = p
	matrix.b = b
	matrix.m = m
}

func GetFlowVelocity() float64 {
	return matrix.v
}

func GetFlowDensity() float64 {
	return matrix.p
}

func GetBladeChord() float64 {
	return matrix.b
}

func GetBladeMass() float64 {
	return matrix.m
}
