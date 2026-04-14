package tests

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	config "masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	graph "masters/internal/graph"
)

const (
	eps = 1e-6
)

// Expected SHA256 hashes to ensure fixtures (config + wolfram reference points) are not modified.
// These were computed from the repository contents at time of test creation.
const (
	expectedHashConfig2Bodies = "70b4db5965084d969b20c3d71d8384c14a82c5c217fcf5b86fd26cc27df25435"
	expectedHashGraphPoints0  = "690bebeb1bbfb4cfd85229ae88c29120a7679c5dd1119bf90597b01decdaca00"
	expectedHashGraphPoints1  = "274e98cd68996df47f8a07ce5e4ea087800a6dfce02b873ba0f05fbbc4710363"
)

type series struct {
	T []float64
	X []float64
}

func readSeries(path string) (series, error) {
	f, err := os.Open(path)
	if err != nil {
		return series{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024), 1024*1024)

	out := series{T: make([]float64, 0), X: make([]float64, 0)}
	for sc.Scan() {
		var t, x float64
		line := sc.Text()
		// Format is: "<t> <x>\n" with fixed decimal formatting.
		if _, err := fmt.Sscanf(line, "%f %f", &t, &x); err != nil {
			return series{}, fmt.Errorf("failed to parse line %q: %w", line, err)
		}
		out.T = append(out.T, t)
		out.X = append(out.X, x)
	}
	if err := sc.Err(); err != nil {
		return series{}, err
	}
	return out, nil
}

func sha256HexOfFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func findModuleRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found while walking up from: ", filepath.Dir(file))
		}
		dir = parent
	}
}

func getNodePosByID(g *graph.Graph, nodeID int) (float64, bool) {
	for _, n := range g.Nodes {
		if n != nil && n.ID == nodeID {
			return n.Position, true
		}
	}
	return 0, false
}

func Test2BodiesMatchesWolframGraphPoints(t *testing.T) {
	root := findModuleRoot(t)

	configPath := filepath.Join(root, "tests", "testdata", "2Bodies", "2Bodies.json")
	ref0Path := filepath.Join(root, "tests", "testdata", "2Bodies", "graph_points0.txt")
	ref1Path := filepath.Join(root, "tests", "testdata", "2Bodies", "graph_points1.txt")

	checkHash := func(path, want string) {
		t.Helper()
		got, err := sha256HexOfFile(path)
		if err != nil {
			t.Fatalf("sha256(%s): %v", path, err)
		}
		if got != want {
			t.Fatalf("fixture modified: %s\nexpected sha256=%s\ngot      sha256=%s", path, want, got)
		}
	}

	checkHash(configPath, expectedHashConfig2Bodies)
	checkHash(ref0Path, expectedHashGraphPoints0)
	checkHash(ref1Path, expectedHashGraphPoints1)

	ref0, err := readSeries(ref0Path)
	if err != nil {
		t.Fatalf("read ref0: %v", err)
	}
	ref1, err := readSeries(ref1Path)
	if err != nil {
		t.Fatalf("read ref1: %v", err)
	}

	if len(ref0.T) == 0 || len(ref0.X) == 0 {
		t.Fatal("ref0 is empty")
	}
	if len(ref0.T) != len(ref1.T) || len(ref0.X) != len(ref1.X) {
		t.Fatalf("reference series length mismatch: ref0=%d, ref1=%d", len(ref0.T), len(ref1.T))
	}

	// Load config for this test explicitly from testdata (do not rely on CONFIG/cwd).
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config fixture: %v", err)
	}
	var cfg config.Graph
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config fixture: %v", err)
	}

	// Build graph manually to avoid touching global config/cache from config package.
	g := graph.NewGraph()
	maxID := -1
	for _, n := range cfg.Nodes {
		if n.ID > maxID {
			maxID = n.ID
		}
	}
	if maxID < 0 {
		t.Fatal("invalid config: empty nodes")
	}
	g.Nodes = make([]*config.Node, maxID+1)
	for i := range cfg.Nodes {
		n := &cfg.Nodes[i]
		if n.IsFixed {
			g.Nodes[n.ID] = graph.NewFixedNode(n.ID, n.Position)
		} else {
			g.Nodes[n.ID] = n
		}
	}
	// Guard: ensure solver doesn't hit nil nodes (computeAccelerations dereferences IsFixed).
	for id := range g.Nodes {
		if g.Nodes[id] == nil {
			g.Nodes[id] = graph.NewFixedNode(id, 0)
		}
	}
	g.NodesNum = len(cfg.Nodes)
	g.Period = 0
	g.FirstNodeID = 0
	g.LastNodeID = 0

	for _, e := range cfg.Edges {
		g.AddEdge(e)
	}

	// solver uses RK4 and dt from config.
	solver := equationsolver.NewGraphSolver(g, cfg.Times.Dt)

	nSteps := int(math.Round((cfg.Times.T-cfg.Times.T0)/cfg.Times.Dt))
	if nSteps <= 0 {
		t.Fatalf("unexpected nSteps: %d", nSteps)
	}

	got0 := make([]float64, 0, nSteps+1)
	got1 := make([]float64, 0, nSteps+1)

	for step := 0; step <= nSteps; step++ {
		tNow := cfg.Times.T0 + float64(step)*cfg.Times.Dt
		// Important: to match `WriteGraphPointsToFiles`, we call Step(t) and only then read positions.
		solver.Step(tNow)

		x0, ok0 := getNodePosByID(g, 0)
		x1, ok1 := getNodePosByID(g, 1)
		if !ok0 || !ok1 {
			t.Fatalf("cannot find node positions for IDs 0 and/or 1 (ok0=%v, ok1=%v)", ok0, ok1)
		}

		got0 = append(got0, x0)
		got1 = append(got1, x1)

		// Compare by index to guarantee "even one mismatch fails".
		refIdx := step
		if refIdx >= len(ref0.T) {
			t.Fatalf("reference shorter than simulation: refLen=%d, simLen=%d", len(ref0.T), nSteps+1)
		}

		if math.Abs(ref0.T[refIdx]-tNow) > 1e-9 || math.Abs(ref1.T[refIdx]-tNow) > 1e-9 {
			t.Fatalf("time mismatch at idx=%d: refT0=%g refT1=%g simT=%g", refIdx, ref0.T[refIdx], ref1.T[refIdx], tNow)
		}
		if math.Abs(ref0.X[refIdx]-x0) > eps {
			t.Fatalf("node0 mismatch at idx=%d t=%g: expected=%g got=%g absErr=%g", refIdx, tNow, ref0.X[refIdx], x0, math.Abs(ref0.X[refIdx]-x0))
		}
		if math.Abs(ref1.X[refIdx]-x1) > eps {
			t.Fatalf("node1 mismatch at idx=%d t=%g: expected=%g got=%g absErr=%g", refIdx, tNow, ref1.X[refIdx], x1, math.Abs(ref1.X[refIdx]-x1))
		}
	}

	// Also guard against reference having extra lines (shouldn't happen).
	if len(ref0.X) != len(got0) || len(ref1.X) != len(got1) {
		t.Fatalf("series length mismatch after run: ref0=%d got0=%d, ref1=%d got1=%d",
			len(ref0.X), len(got0), len(ref1.X), len(got1),
		)
	}
}

