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
	var bodiesConds config.BodiesConds
	var timeConds config.TimeConds

	if err := config.CondsInit(&bodiesConds, &timeConds); err != nil {
		log.Errorf("%s", err)

		os.Exit(1)
	}

	if bodiesConds.IsCoupled {
		equationsolver.SolverCoupled(&bodiesConds, &timeConds)
	} else {
		equationsolver.Solver(&bodiesConds, &timeConds)
	}
}
