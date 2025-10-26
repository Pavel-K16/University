package main

import (
	"fmt"
	"math"
	"masters/internal/config"
	equationsolver "masters/internal/equationSolver"
)

func main() {
	fmt.Println("=== Детальный тест с логированием ===")
	
	graph := config.NewGraph()
	ground := config.NewFixedNode(0, 0.0)
	graph.AddNode(ground)
	
	// Одно тело на пружине с демпфером
	body := config.NewMovableNode(1, 1.0, 1.0, 0.0, 0.0, 0.0)
	graph.AddNode(body)
	graph.AddEdge(0, 1, 1.0, 0.5, 0.0)
	
	dt := 0.01
	solver := equationsolver.NewGraphSolver(graph, dt)
	
	// Проверяем первые несколько шагов
	fmt.Println("\nШаг  | t      | x       | v       | F       | аналит.x")
	fmt.Println("-----|--------|---------|---------|---------|---------")
	
	gamma := 0.5 / (2.0 * 1.0)
	omega := math.Sqrt(1.0 - gamma*gamma)
	
	for step := 0; step < 10; step++ {
		t := float64(step) * dt
		solver.Step(t)
		
		x := body.Position
		v := body.Velocity
		F := graph.NetForce(1)
		analytical := math.Exp(-gamma*t) * math.Cos(omega*t)
		
		fmt.Printf("%4d | %.4f | %7.4f | %7.4f | %7.4f | %7.4f\n", step, t, x, v, F, analytical)
	}
	
	// Проверяем силы подробнее
	fmt.Println("\n=== Детальный анализ сил ===")
	fmt.Println("Для первого шага:")
	
	// Восстанавливаем
	body.Position = 1.0
	body.Velocity = 0.0
	graph2 := config.NewGraph()
	graph2.AddNode(ground)
	body2 := config.NewMovableNode(1, 1.0, 1.0, 0.0, 0.0, 0.0)
	graph2.AddNode(body2)
	graph2.AddEdge(0, 1, 1.0, 0.5, 0.0)
	
	fmt.Println("Начальное состояние:")
	fmt.Printf("  Позиция: %.6f\n", body2.Position)
	fmt.Printf("  Скорость: %.6f\n", body2.Velocity)
	
	F := graph2.NetForce(1)
	fmt.Printf("\nЧистая сила: %.6f\n", F)
	
	// Проверяем вклад каждого ребра
	for _, edge := range body2.Edges {
		targetNode := graph2.Nodes[edge.TargetID]
		dx := body2.Position - targetNode.Position - edge.Rest
		dv := body2.Velocity - targetNode.Velocity
		springF := -edge.K * dx
		damperF := -edge.D * dv
		fmt.Printf("\nРебро с узлом %d:\n", edge.TargetID)
		fmt.Printf("  dx = %.6f (позиция - целевая позиция)\n", dx)
		fmt.Printf("  dv = %.6f (скорость - целевая скорость)\n", dv)
		fmt.Printf("  Сила пружины: %.6f\n", springF)
		fmt.Printf("  Сила демпфера: %.6f\n", damperF)
		fmt.Printf("  Суммарная сила от ребра: %.6f\n", springF+damperF)
	}
}

