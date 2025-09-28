package config

type BodyInterface interface {
	GetID() int
	GetMass() float64
	GetPosition() float64
	GetVelocity() float64
	GetStiffness() float64
	GetDamping() float64
	GetCouplings() []Coupling
	SetPosition(float64)
	SetVelocity(float64)
}

type SimpleBody struct {
	ID        int
	Mass      float64
	Position  float64
	Velocity  float64
	K         float64
	D         float64
	Couplings []Coupling
}

func (b *SimpleBody) GetID() int               { return b.ID }
func (b *SimpleBody) GetMass() float64         { return b.Mass }
func (b *SimpleBody) GetPosition() float64     { return b.Position }
func (b *SimpleBody) GetVelocity() float64     { return b.Velocity }
func (b *SimpleBody) GetStiffness() float64    { return b.K }
func (b *SimpleBody) GetDamping() float64      { return b.D }
func (b *SimpleBody) GetCouplings() []Coupling { return b.Couplings }
func (b *SimpleBody) SetPosition(x float64)    { b.Position = x }
func (b *SimpleBody) SetVelocity(v float64)    { b.Velocity = v }

type Force interface {
	// Возвращает вклад силы на целевое тело (индекс bodyIdx) с учётом всех тел (bodies) в момент времени t
	Calculate(bodyIdx int, bodies []BodyInterface, t float64) float64
	// Список индексов тел, на которые эта сила влияет (для индексации в реестре)
	Targets() []int
}

type ForceRegistry struct {
	byBody map[int][]Force
}

func NewForceRegistry() *ForceRegistry {
	return &ForceRegistry{byBody: make(map[int][]Force)}
}

func (r *ForceRegistry) Index(forces []Force) {
	for _, f := range forces {
		for _, idx := range f.Targets() {
			r.byBody[idx] = append(r.byBody[idx], f)
		}
	}
}

func (r *ForceRegistry) NetForce(bodyIdx int, bodies []BodyInterface, t float64) float64 {
	var sum float64
	for _, f := range r.byBody[bodyIdx] {
		sum += f.Calculate(bodyIdx, bodies, t)
	}
	return sum
}

func (r *ForceRegistry) GetForceCount(bodyIdx int) int {
	return len(r.byBody[bodyIdx])
}

func (r *ForceRegistry) GetForces(bodyIdx int) []Force {
	return r.byBody[bodyIdx]
}

type SpringForce struct {
	i, j int
	k    float64
	rest float64
}

func (f *SpringForce) Targets() []int { return []int{f.i, f.j} }
func (f *SpringForce) Calculate(bodyIdx int, b []BodyInterface, _ float64) float64 {
	xi, xj := b[f.i].GetPosition(), b[f.j].GetPosition()
	disp := (xi - xj) - f.rest
	F := -f.k * disp
	if bodyIdx == f.i {
		return F
	}
	if bodyIdx == f.j {
		return -F
	}
	return 0
}

type DamperForce struct {
	i, j int
	d    float64
}

func (f *DamperForce) Targets() []int { return []int{f.i, f.j} }
func (f *DamperForce) Calculate(bodyIdx int, b []BodyInterface, _ float64) float64 {
	vi, vj := b[f.i].GetVelocity(), b[f.j].GetVelocity()
	F := -f.d * (vi - vj)
	if bodyIdx == f.i {
		return F
	}
	if bodyIdx == f.j {
		return -F
	}
	return 0
}

// GroundSpringForce — собственная жесткость тела относительно "земли": F = -k_i * x_i
type GroundSpringForce struct{ i int }

func (f *GroundSpringForce) Targets() []int { return []int{f.i} }
func (f *GroundSpringForce) Calculate(bodyIdx int, b []BodyInterface, _ float64) float64 {
	if bodyIdx != f.i {
		return 0
	}
	ki := b[f.i].GetStiffness()
	return -ki * b[f.i].GetPosition()
}

// GroundDamperForce — собственное демпфирование тела относительно "земли": F = -d_i * v_i
type GroundDamperForce struct{ i int }

func (f *GroundDamperForce) Targets() []int { return []int{f.i} }
func (f *GroundDamperForce) Calculate(bodyIdx int, b []BodyInterface, _ float64) float64 {
	if bodyIdx != f.i {
		return 0
	}
	di := b[f.i].GetDamping()
	// При отрицательном демпфировании сила должна быть положительной (ускоряющей)
	return -di * b[f.i].GetVelocity()
}

// AssembleForces собирает список сил из собственных свойств тел (k_i, d_i)
// и матриц связей k_ij, d_ij (верхнетреугольные или полные, вне диагонали).
// Ожидается, что len(kij)==N и len(kij[i])==N, то же для dij; нули игнорируются.
func AssembleForces(bodies []BodyInterface, kij [][]float64, dij [][]float64) []Force {
	n := len(bodies)
	forces := make([]Force, 0, n*3)
	// Собственные силы
	for i := 0; i < n; i++ {
		if bodies[i].GetStiffness() != 0 {
			forces = append(forces, &GroundSpringForce{i: i})
		}
		if bodies[i].GetDamping() != 0 {
			forces = append(forces, &GroundDamperForce{i: i})
		}
	}
	// Связи между телами
	if kij != nil {
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				k := kij[i][j]
				if k != 0 {
					forces = append(forces, &SpringForce{i: i, j: j, k: k, rest: 0})
				}
			}
		}
	}
	if dij != nil {
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				d := dij[i][j]
				if d != 0 {
					forces = append(forces, &DamperForce{i: i, j: j, d: d})
				}
			}
		}
	}
	return forces
}

// Coupling — связь текущего тела с телом J. Коэффициенты могут быть любого знака.
type Coupling struct {
	J    int     // индекс другого тела
	Kij  float64 // жёсткость связи
	Dij  float64 // демпфирование связи
	Rest float64 // длина покоя (обычно 0)
}

// AssembleForcesFromBodies — собирает силы только из «вшитых» связей тел и их собственных k,d.
// Если обе стороны пары задали связь, коэффициенты суммируются (Kpair = k_ij + k_ji и т.п.).
func AssembleForcesFromBodies(bodies []BodyInterface) []Force {
	n := len(bodies)
	forces := make([]Force, 0, n*3)

	// собственные силы относительно земли
	for i := 0; i < n; i++ {
		if bodies[i].GetStiffness() != 0 {
			forces = append(forces, &GroundSpringForce{i: i})
		}
		if bodies[i].GetDamping() != 0 {
			forces = append(forces, &GroundDamperForce{i: i})
		}
	}

	type pair struct{ a, b int }
	norm := func(i, j int) pair {
		if i < j {
			return pair{i, j}
		}
		return pair{j, i}
	}

	sumK := make(map[pair]float64)
	sumD := make(map[pair]float64)
	rest := make(map[pair]float64)

	for i := 0; i < n; i++ {
		for _, c := range bodies[i].GetCouplings() {
			if c.J < 0 || c.J >= n || c.J == i {
				continue
			}
			key := norm(i, c.J)
			sumK[key] += c.Kij
			sumD[key] += c.Dij
			if c.Rest != 0 {
				rest[key] = c.Rest
			}
		}
	}

	for k, Kpair := range sumK {
		if Kpair != 0 {
			forces = append(forces, &SpringForce{i: k.a, j: k.b, k: Kpair, rest: rest[k]})
		}
	}
	for k, Dpair := range sumD {
		if Dpair != 0 {
			forces = append(forces, &DamperForce{i: k.a, j: k.b, d: Dpair})
		}
	}

	return forces
}
