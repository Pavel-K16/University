package config

import (
	"encoding/json"
	"fmt"
	"masters/internal/logger"
	"os"
)

type Graph struct {
	Times         Times   `json:"times"`
	Aero          Aero    `json:"aeroDynamicForce,omitempty"`
	Nodes         []Node  `json:"nodes"`
	Edges         []Edge  `json:"edges"`
	LastFirstDist float64 `json:"lastFirstDist"`
}

type Times struct {
	T0 float64 `json:"t0"`
	T  float64 `json:"t"`
	Dt float64 `json:"dt"`
}

type Aero struct {
	Enabled bool    `json:"enabled,omitempty"`
	Scale   float64 `json:"scale,omitempty"`
	V       float64 `json:"v,omitempty"`
	Pho     float64 `json:"rho,omitempty"`
	B       float64 `json:"b,omitempty"`
	M       float64 `json:"m,omitempty"`
}

type Node struct {
	ID       int     `json:"id"`
	IsFixed  bool    `json:"fixed"`
	Mass     float64 `json:"mass,omitempty"`
	Position float64 `json:"position"`
	Velocity float64 `json:"velocity,omitempty"`
	K        float64
	D        float64
	Edges    []Edge
}

type Edge struct {
	FromID   int     `json:"from"`
	TargetID int     `json:"to"`
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

// ReadGraphConfig читает JSON по пути ConfigPath(), возвращает новый *Graph.
// У каждого вызова — свой экземпляр узлов в памяти; это безопасно для параллельных
// прогонов, если каждая горутина вызывает ReadGraphConfig() отдельно (или иначе
// имеет свою копию конфигурации).
func ReadGraphConfig() (*Graph, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	return ParseGraphConfig(data)
}

// ParseGraphConfig разбирает JSON графа без чтения файла (удобно для тестов).
func ParseGraphConfig(data []byte) (*Graph, error) {
	var cfg Graph
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal graph config: %w", err)
	}
	if os.Getenv("CONFIG") == "2Bodies" {
		Write2BodiesParamsToTxt(&cfg)
	}
	return &cfg, nil
}

type nodeOwnParams struct {
	k float64
	d float64
}

func Write2BodiesParamsToTxt(cnf *Graph) {
	nodes := cnf.Nodes
	edges := cnf.Edges

	var fixedNodeID int

	var coupledK, coupledD float64

	nodesOwnParams := make(map[int]nodeOwnParams)

	for _, node := range nodes {
		if node.IsFixed {
			fixedNodeID = node.ID
			break
		}
	}

	for _, edge := range edges {
		if edge.TargetID == fixedNodeID {

			nodesOwnParams[edge.FromID] = nodeOwnParams{
				k: edge.K,
				d: edge.D,
			}
		}

		if edge.FromID != fixedNodeID && edge.TargetID != fixedNodeID {
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
		if node.IsFixed {
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
