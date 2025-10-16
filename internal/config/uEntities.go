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

// FixedBody — жёстко зафиксированный узел (неподвижный)
type FixedBody struct {
	ID       int
	Position float64 // фиксированная позиция
}

func (f *FixedBody) GetID() int               { return f.ID }
func (f *FixedBody) GetMass() float64         { return 0 } // масса не важна для неподвижного узла
func (f *FixedBody) GetPosition() float64     { return f.Position }
func (f *FixedBody) GetVelocity() float64     { return 0 } // скорость всегда 0
func (f *FixedBody) GetStiffness() float64    { return 0 } // собственных сил нет
func (f *FixedBody) GetDamping() float64      { return 0 }
func (f *FixedBody) GetCouplings() []Coupling { return nil } // связей нет
func (f *FixedBody) SetPosition(float64)      { /* игнорируем - узел неподвижен */ }
func (f *FixedBody) SetVelocity(float64)      { /* игнорируем - узел неподвижен */ }

// RigidlyConnectedBody — узел, жёстко связанный с мастер-узлом (двигается вместе с ним)
type RigidlyConnectedBody struct {
	ID           int
	Mass         float64
	Position     float64
	Velocity     float64
	K            float64
	D            float64
	Couplings    []Coupling
	MasterNodeID int     // ID узла-мастера, за которым следует этот узел
	Offset       float64 // смещение относительно мастер-узла
}

func (r *RigidlyConnectedBody) GetID() int               { return r.ID }
func (r *RigidlyConnectedBody) GetMass() float64         { return r.Mass }
func (r *RigidlyConnectedBody) GetPosition() float64     { return r.Position }
func (r *RigidlyConnectedBody) GetVelocity() float64     { return r.Velocity }
func (r *RigidlyConnectedBody) GetStiffness() float64    { return r.K }
func (r *RigidlyConnectedBody) GetDamping() float64      { return r.D }
func (r *RigidlyConnectedBody) GetCouplings() []Coupling { return r.Couplings }
func (r *RigidlyConnectedBody) SetPosition(pos float64)  { r.Position = pos }
func (r *RigidlyConnectedBody) SetVelocity(vel float64)  { r.Velocity = vel }
func (r *RigidlyConnectedBody) GetMasterNodeID() int     { return r.MasterNodeID }
func (r *RigidlyConnectedBody) GetOffset() float64       { return r.Offset }

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
	I, J int
	K    float64
	Rest float64
}

func (f *SpringForce) Targets() []int { return []int{f.I, f.J} }
func (f *SpringForce) Calculate(bodyIdx int, b []BodyInterface, _ float64) float64 {
	xi, xj := b[f.I].GetPosition(), b[f.J].GetPosition()
	disp := (xi - xj) - f.Rest
	F := -f.K * disp
	if bodyIdx == f.I {
		return F
	}
	if bodyIdx == f.J {
		return -F
	}
	return 0
}

type DamperForce struct {
	I, J int
	D    float64
}

func (f *DamperForce) Targets() []int { return []int{f.I, f.J} }
func (f *DamperForce) Calculate(bodyIdx int, b []BodyInterface, _ float64) float64 {
	vi, vj := b[f.I].GetVelocity(), b[f.J].GetVelocity()
	F := -f.D * (vi - vj)
	if bodyIdx == f.I {
		return F
	}
	if bodyIdx == f.J {
		return -F
	}
	return 0
}

// GroundSpringForce — собственная жесткость тела относительно "земли": F = -k_i * x_i
type GroundSpringForce struct{ I int }

func (f *GroundSpringForce) Targets() []int { return []int{f.I} }
func (f *GroundSpringForce) Calculate(bodyIdx int, b []BodyInterface, _ float64) float64 {
	if bodyIdx != f.I {
		return 0
	}
	ki := b[f.I].GetStiffness()
	return -ki * b[f.I].GetPosition()
}

// GroundDamperForce — собственное демпфирование тела относительно "земли": F = -d_i * v_i
type GroundDamperForce struct{ I int }

func (f *GroundDamperForce) Targets() []int { return []int{f.I} }
func (f *GroundDamperForce) Calculate(bodyIdx int, b []BodyInterface, _ float64) float64 {
	if bodyIdx != f.I {
		return 0
	}
	di := b[f.I].GetDamping()
	// При отрицательном демпфировании сила должна быть положительной (ускоряющей)
	return -di * b[f.I].GetVelocity()
}

// PlatformSpringForce — сила пружины метронома относительно платформы
type PlatformSpringForce struct {
	I           int     // индекс метронома
	PlatformIdx int     // индекс платформы
	K           float64 // коэффициент жёсткости
	Rest        float64 // длина в покое
}

func (f *PlatformSpringForce) Calculate(bodyIdx int, bodies []BodyInterface, t float64) float64 {
	if bodyIdx != f.I {
		return 0
	}
	// Сила относительно платформы: -k * (позиция_метронома - позиция_платформы - rest)
	return -f.K * (bodies[f.I].GetPosition() - bodies[f.PlatformIdx].GetPosition() - f.Rest)
}

func (f *PlatformSpringForce) Targets() []int {
	return []int{f.I}
}

// PlatformDamperForce — сила демпфера метронома относительно платформы
type PlatformDamperForce struct {
	I           int     // индекс метронома
	PlatformIdx int     // индекс платформы
	D           float64 // коэффициент демпфирования
}

func (f *PlatformDamperForce) Calculate(bodyIdx int, bodies []BodyInterface, t float64) float64 {
	if bodyIdx != f.I {
		return 0
	}
	// Сила демпфера относительно платформы: -d * (скорость_метронома - скорость_платформы)
	return -f.D * (bodies[f.I].GetVelocity() - bodies[f.PlatformIdx].GetVelocity())
}

func (f *PlatformDamperForce) Targets() []int {
	return []int{f.I}
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
			forces = append(forces, &GroundSpringForce{I: i})
		}
		if bodies[i].GetDamping() != 0 {
			forces = append(forces, &GroundDamperForce{I: i})
		}
	}
	// Связи между телами
	if kij != nil {
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				k := kij[i][j]
				if k != 0 {
					forces = append(forces, &SpringForce{I: i, J: j, K: k, Rest: 0})
				}
			}
		}
	}
	if dij != nil {
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				d := dij[i][j]
				if d != 0 {
					forces = append(forces, &DamperForce{I: i, J: j, D: d})
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
			forces = append(forces, &GroundSpringForce{I: i})
		}
		if bodies[i].GetDamping() != 0 {
			forces = append(forces, &GroundDamperForce{I: i})
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
			forces = append(forces, &SpringForce{I: k.a, J: k.b, K: Kpair, Rest: rest[k]})
		}
	}
	for k, Dpair := range sumD {
		if Dpair != 0 {
			forces = append(forces, &DamperForce{I: k.a, J: k.b, D: Dpair})
		}
	}

	return forces
}

// AutoAssembleForcesFromBodies — автоматически создаёт силы на основе конфигурации тел
func AutoAssembleForcesFromBodies(bodies []BodyInterface) []Force {
	forces := make([]Force, 0)

	// Проходим по всем телам и создаём силы на основе их свойств
	for i, body := range bodies {
		// Пропускаем фиксированные тела
		if _, isFixed := body.(*FixedBody); isFixed {
			continue
		}

		// Собственные силы тела (если есть)
		if body.GetStiffness() != 0 {
			forces = append(forces, &GroundSpringForce{I: i})
		}
		if body.GetDamping() != 0 {
			forces = append(forces, &GroundDamperForce{I: i})
		}

		// Обрабатываем связи тела
		for _, coupling := range body.GetCouplings() {
			j := coupling.J

			// Проверяем, что связь валидна
			if j >= len(bodies) || j == i {
				continue
			}

			// Создаём силы связи между телами
			if coupling.Kij != 0 {
				forces = append(forces, &SpringForce{
					I: i, J: j, K: coupling.Kij, Rest: coupling.Rest,
				})
			}
			if coupling.Dij != 0 {
				forces = append(forces, &DamperForce{
					I: i, J: j, D: coupling.Dij,
				})
			}
		}
	}

	return forces
}
