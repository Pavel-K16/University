package main

import (
	"masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	"masters/internal/logger"
	"os"
)

var (
	log = logger.LoggerInit()
)

func main() {
	var conds config.InitialConds
	var coupledConds config.InitialCondsCoupled

	if err := config.CondsInit(&conds); err != nil {
		log.Errorf("%s", err)

		os.Exit(1)
	}

	if err := config.CoupledCondsInit(&coupledConds); err != nil {
		log.Errorf("%s", err)

		os.Exit(1)
	}

	//equationsolver.Solver(&conds)
	equationsolver.SolverCoupled(&coupledConds, &conds)
}
