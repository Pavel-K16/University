package config

import (
	"encoding/json"
	"errors"
	g "masters/internal/graph"
	"masters/internal/logger"
	"os"
)

type GraphConfig struct {
	Nodes []NodeConfig `json:"nodes"`
	Edges []EdgeConfig `json:"edges"`
}

type NodeConfig struct {
	ID       int     `json:"id"`
	Fixed    bool    `json:"fixed"`
	Mass     float64 `json:"mass,omitempty"`
	Position float64 `json:"position"`
	Velocity float64 `json:"velocity,omitempty"`
}

type EdgeConfig struct {
	From int     `json:"from"`
	To   int     `json:"to"`
	K    float64 `json:"k,omitempty"`
	D    float64 `json:"d,omitempty"`
	Rest float64 `json:"rest,omitempty"`
}

const (
	confJsonPath = "../internal/config/confs/conf.json"
)

var (
	log = logger.LoggerInit()
)

func loadGraphConfig() (*GraphConfig, error) {
	data, err := os.ReadFile(confJsonPath)
	if err != nil {
		return nil, err
	}

	var cfg GraphConfig

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func CreateGraph(graph *g.Graph) error {
	cnf, err := loadGraphConfig()
	if err != nil {
		log.Errorf("Error: %s", err)

		return err
	}

	if cnf == nil {
		log.Errorf("Got empty config")

		return errors.New("Got empty config")
	}

	num := len(cnf.Nodes) - 1

	for _, node := range cnf.Nodes {
		if node.Fixed {
			fixedNode := g.NewFixedNode(node.ID, node.Position)
			graph.AddNode(fixedNode)

			continue
		}

		metronome0 := g.NewMovableNode(node.ID,
			node.Mass,
			node.Position,
			node.Velocity,
			0.0,
			0.0,
		)
		graph.AddNode(metronome0)
	}

	for _, edge := range cnf.Edges {
		if edge.From > num || edge.To > num {
			continue
		}

		graph.AddEdge(edge.From, edge.To, edge.K, edge.D, edge.Rest)
	}

	return nil
}
