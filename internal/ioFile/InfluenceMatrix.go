package iofile

import "masters/internal/config"

type influenceKoefMatrix struct {
	matrix [][]int
}

var (
	matrix   influenceKoefMatrix
	nodesNum []int
)

func InitInfluenceKoefMatrix(graph *config.Graph) {
	nodesNum = graph.NodesNumbers()

	matrix.matrix = make([][]int, len(nodesNum))
	for i := range nodesNum {
		matrix.matrix[i] = make([]int, len(nodesNum))
	}

	for i := range nodesNum {
		for j := range nodesNum {
			matrix.matrix[i][j] = 0
		}
	}
}

func GetInfluenceKoef(num1, num2 int) int {
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

func SetInfluenceKoef(num1, num2, value int) {
	if num1 >= len(matrix.matrix) || num2 >= len(matrix.matrix[0]) {
		log.Errorf("num1 or num2 is out of range: %d, %d", num1, num2)

		return
	}

	mapNum := make(map[int]int)
	for i, num := range nodesNum {
		mapNum[num] = i
	}

	matrix.matrix[mapNum[num1]][mapNum[num2]] = value
}
