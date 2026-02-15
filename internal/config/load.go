package config

import (
	"encoding/json"
	g "masters/internal/graph"
	"masters/internal/logger"
	"os"
)

type GraphConfig struct {
	Times TimesConfig  `json:"times"`
	Aero  AeroConfig   `json:"aeroDynamicForce"`
	Nodes []NodeConfig `json:"nodes"`
	Edges []EdgeConfig `json:"edges"`
}

type TimesConfig struct {
	T0 float64 `json:"t0"`
	T  float64 `json:"t"`
	Dt float64 `json:"dt"`
}

type AeroConfig struct {
	Enabled bool    `json:"enabled"`
	Scale   float64 `json:"scale"`
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
	confDir     = "../internal/config/confs" // относительно cwd при запуске из scripts/
	defaultConf = "conf"
)

var gConfig *GraphConfig

func ConfigPath() string {
	name := os.Getenv("CONFIG")
	if name == "" {
		name = defaultConf
	}
	return confDir + "/" + name + ".json"
}

var (
	log = logger.LoggerInit()
)

func LoadGraphConfig() error {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg GraphConfig

	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	gConfig = &cfg

	return nil
}

func CreateGraph(graph *g.Graph) error {
	if gConfig == nil {
		log.Warningf("Got empty config")
		if err := LoadGraphConfig(); err != nil {
			log.Errorf("Error Load Config: %s", err)
		}
	}

	cnf := gConfig

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

func GetConfig() *GraphConfig {
	if gConfig == nil {
		LoadGraphConfig()
	}

	return gConfig
}
