package main

import (
	"fmt"
	"masters/internal/config"
	equationsolver "masters/internal/equationSolver"
)

func main() {
	//os.Chdir("..")
	//logger.LoggerInit()

	var conds config.InitialConds
	config.CondsInit(&conds)
	fmt.Println("Conds:", conds)

	equationsolver.Solver(&conds)
}
