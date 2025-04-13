package main

import (
	"fmt"
	"masters/config"
)

func main() {
	fmt.Println("wfwef")
	var popa config.Jopa
	config.Init(&popa)
	fmt.Println(popa)
}
