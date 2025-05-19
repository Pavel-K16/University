package main

import (
	"fmt"
	"masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	"masters/internal/logger"
	"os"
)

var (
	log = logger.LoggerInit()
)

func main() {
	var bodiesConds config.BodiesConds
	var timeConds config.TimeConds

	if err := config.CondsInit(&bodiesConds, &timeConds); err != nil {
		log.Errorf("%s", err)

		os.Exit(1)
	}

	coupledFile, _ := os.OpenFile("../wolfram/paramsAndPoints/coupled.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	fmt.Fprintf(coupledFile, "%.10f\n", bodiesConds.ConnParams.D12)
	fmt.Fprintf(coupledFile, "%.10f\n", bodiesConds.ConnParams.K12)

	if bodiesConds.IsCoupled {
		equationsolver.SolverCoupled(&bodiesConds, &timeConds)
	} else {
		equationsolver.Solver(&bodiesConds, &timeConds)
	}
}
