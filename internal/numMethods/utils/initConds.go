package utils

import (
	"masters/internal/config"
	"masters/internal/logger"
)

var (
	log = logger.LoggerInit()
)

func InitConds4F(conds *config.InitialConds) (float64, float64, float64) {
	log.Tracef("InitConds4F.initConds.go")

	k := conds.K
	m := conds.M
	d := conds.D

	return k, m, d
}

func InitConds4aS(conds *config.InitialConds) (float64, float64, float64, float64, float64) {
	log.Tracef("InitConds4F.initConds.go")

	k, m, d := InitConds4F(conds)
	x0 := conds.X0
	v0 := conds.V0

	return k, m, d, x0, v0
}

func InitConds4Solver(conds *config.InitialConds) (float64, float64, float64) {
	log.Tracef("InitConds4F.initConds.go")

	t0 := conds.T0
	t := conds.T
	tau := conds.Tau

	return t0, t, tau
}
