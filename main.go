package main

import (
	"fmt"
	"masters/config"
	"masters/logger"
)

func main() {
	//os.Chdir("..")
	logger.LoggerInit()

	var conds config.InitialConds
	config.CondsInit(&conds)
	fmt.Println(conds)
}
