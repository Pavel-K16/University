package inmemory

type Point struct {
	T float64
	X float64
}

type PointsStore struct {
	points map[int][]Point
}

func NewPointsStore() *PointsStore {
	return &PointsStore{points: make(map[int][]Point)}
}

func (s *PointsStore) AddPoint(nodeID int, point Point) {
	s.points[nodeID] = append(s.points[nodeID], point)
}

func (s *PointsStore) GetPoints(nodeID int) []Point {
	return s.points[nodeID]
}
