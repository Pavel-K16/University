package equationsolver

import (
	"masters/internal/config"
)

type USolver struct {
	bodies []config.BodyInterface
	reg    *config.ForceRegistry
	dt     float64
}

func NewSolver(bodies []config.BodyInterface, reg *config.ForceRegistry, dt float64) *USolver {
	return &USolver{bodies: bodies, reg: reg, dt: dt}
}

// делаем снимок состояния, чтобы все ускорения считались на одном слое
func (s *USolver) snapshot() []config.BodyInterface {
	snap := make([]config.BodyInterface, len(s.bodies))
	for i, b := range s.bodies {
		snap[i] = &config.SimpleBody{
			ID:       b.GetID(),
			Mass:     b.GetMass(),
			Position: b.GetPosition(),
			Velocity: b.GetVelocity(),
			K:        b.GetStiffness(),
			D:        b.GetDamping(),
		}
	}
	return snap
}

func (s *USolver) Step(t float64) {
	// Метод Рунге-Кутты 4-го порядка для лучшей точности
	for i := range s.bodies {
		// k1: используем текущие значения
		snap1 := s.snapshot()
		k1v := snap1[i].GetVelocity()
		k1a := s.reg.NetForce(i, snap1, t) / snap1[i].GetMass()

		// k2: используем промежуточные значения (текущие + k1/2)
		snap2 := s.snapshot()
		for j := range snap2 {
			if j == i {
				snap2[j] = &config.SimpleBody{
					ID: snap2[j].GetID(), Mass: snap2[j].GetMass(),
					Position: snap2[j].GetPosition() + s.dt*k1v/2,
					Velocity: snap2[j].GetVelocity() + s.dt*k1a/2,
					K:        snap2[j].GetStiffness(), D: snap2[j].GetDamping(),
				}
			}
		}
		k2v := snap2[i].GetVelocity()
		k2a := s.reg.NetForce(i, snap2, t+s.dt/2) / snap2[i].GetMass()

		// k3: используем промежуточные значения (текущие + k2/2)
		snap3 := s.snapshot()
		for j := range snap3 {
			if j == i {
				snap3[j] = &config.SimpleBody{
					ID: snap3[j].GetID(), Mass: snap3[j].GetMass(),
					Position: snap3[j].GetPosition() + s.dt*k2v/2,
					Velocity: snap3[j].GetVelocity() + s.dt*k2a/2,
					K:        snap3[j].GetStiffness(), D: snap3[j].GetDamping(),
				}
			}
		}
		k3v := snap3[i].GetVelocity()
		k3a := s.reg.NetForce(i, snap3, t+s.dt/2) / snap3[i].GetMass()

		// k4: используем промежуточные значения (текущие + k3)
		snap4 := s.snapshot()
		for j := range snap4 {
			if j == i {
				snap4[j] = &config.SimpleBody{
					ID: snap4[j].GetID(), Mass: snap4[j].GetMass(),
					Position: snap4[j].GetPosition() + s.dt*k3v,
					Velocity: snap4[j].GetVelocity() + s.dt*k3a,
					K:        snap4[j].GetStiffness(), D: snap4[j].GetDamping(),
				}
			}
		}
		k4v := snap4[i].GetVelocity()
		k4a := s.reg.NetForce(i, snap4, t+s.dt) / snap4[i].GetMass()

		// Обновляем позицию и скорость по формуле РК4
		newPos := s.bodies[i].GetPosition() + s.dt/6*(k1v+2*k2v+2*k3v+k4v)
		newVel := s.bodies[i].GetVelocity() + s.dt/6*(k1a+2*k2a+2*k3a+k4a)

		// Проверка на стабильность (предотвращение переполнения)
		if newPos > 1e6 || newPos < -1e6 || newVel > 1e6 || newVel < -1e6 {
			// Если значения слишком большие, используем более консервативный шаг
			newPos = s.bodies[i].GetPosition() + s.dt*k1v
			newVel = s.bodies[i].GetVelocity() + s.dt*k1a
		}

		s.bodies[i].SetPosition(newPos)
		s.bodies[i].SetVelocity(newVel)
	}
}
