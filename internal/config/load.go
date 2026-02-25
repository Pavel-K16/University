package config

import (
	"encoding/json"
	"fmt"
	g "masters/internal/graph"
	"masters/internal/logger"
	"os"
)

type GraphConfig struct {
	Times TimesConfig  `json:"times"`
	Aero  AeroConfig   `json:"aeroDynamicForce,omitempty"`
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
	V       float64 `json:"v"`
	Pho     float64 `json:"rho"`
	B       float64 `json:"b"`
	M       float64 `json:"m"`
}

type NodeConfig struct {
	ID       int     `json:"id"`
	Fixed    bool    `json:"fixed"`
	Mass     float64 `json:"mass,omitempty"`
	Position float64 `json:"position"`
	Velocity float64 `json:"velocity,omitempty"`
	K        float64
	D        float64
}

type EdgeConfig struct {
	From     int     `json:"from"`
	To       int     `json:"to"`
	K        float64 `json:"k,omitempty"`
	D        float64 `json:"d,omitempty"`
	Rest     float64 `json:"rest,omitempty"`
	Periodic bool    `json:"periodic,omitempty"`
}

const (
	confDir               = "../internal/config/confs" // относительно cwd при запуске из scripts/
	defaultConf           = "conf"
	paramsFilePathTmpl    = "../wolfram/paramsAndPoints/params%s.txt"
	timeFilePath          = "../wolfram/paramsAndPoints/time.txt"
	coupledParamsFilePath = "../wolfram/paramsAndPoints/coupledParams.txt"
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

	name := os.Getenv("CONFIG")
	if name == "2Bodies" {
		Write2BodiesParamsToTxt(&cfg)
	}

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

		graphNode := g.NewMovableNode(node.ID,
			node.Mass,
			node.Position,
			node.Velocity,
			0.0,
			0.0,
		)
		graph.AddNode(graphNode)
	}

	for _, edge := range cnf.Edges {
		if edge.From > num || edge.To > num {
			continue
		}

		graph.AddEdge(edge.From, edge.To, edge.K, edge.D, edge.Rest, edge.Periodic)
	}

	return nil
}

func GetConfig() *GraphConfig {
	if gConfig == nil {
		LoadGraphConfig()
	}

	return gConfig
}

type nodeOwnParams struct {
	k float64
	d float64
}

func Write2BodiesParamsToTxt(cnf *GraphConfig) {
	nodes := cnf.Nodes
	edges := cnf.Edges

	var fixedNodeID int

	var coupledK, coupledD float64

	nodesOwnParams := make(map[int]nodeOwnParams)

	for _, node := range nodes {
		if node.Fixed {
			fixedNodeID = node.ID
			break
		}
	}

	for _, edge := range edges {
		if edge.To == fixedNodeID {

			nodesOwnParams[edge.From] = nodeOwnParams{
				k: edge.K,
				d: edge.D,
			}
		}

		if edge.From != fixedNodeID && edge.To != fixedNodeID {
			coupledK = edge.K
			coupledD = edge.D
		}
	}

	coupledParamsFile, _ := os.OpenFile(coupledParamsFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	fmt.Fprintf(coupledParamsFile, "%.10f %.10f\n", coupledK, coupledD)
	coupledParamsFile.Close()

	timeParamsFile, _ := os.OpenFile(timeFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	fmt.Fprintf(timeParamsFile, "%.10f %.10f %.10f\n", cnf.Times.T, cnf.Times.T0, cnf.Times.Dt)
	timeParamsFile.Close()

	for _, node := range nodes {
		if node.Fixed {
			continue
		}

		params := nodesOwnParams[node.ID]
		k := params.k
		d := params.d

		paramsFile, _ := os.OpenFile(fmt.Sprintf(paramsFilePathTmpl, fmt.Sprintf("%d", node.ID)), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		fmt.Fprintf(paramsFile, "%.10f %.10f %.10f %.10f %.10f\n", node.Position, node.Mass, node.Velocity, k, d)
		paramsFile.Close()
	}
}
