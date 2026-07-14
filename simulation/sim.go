package simulation

import (
	"math"
	"math/rand"
)

const (
	numSubSteps int     = 1
	gravity     float64 = -9.81 //ms^-2

	Width      float64 = 4   //m
	Height     float64 = 3   //m
	resolution float64 = 100 // total cells vertically
	gridSize   float64 = Height / resolution

	Radius float64 = 5
)

type Particle struct {
	pos [2]float64
	vel [2]float64
}

func (p Particle) GetPos() (x, y float64) {
	return p.pos[0], p.pos[1]
}

// type CellType int

// const (
// 	Air CellType = iota
// 	Water
// 	Solid
// )

type cell struct {
	// cellType CellType
	coord [2]int
	u, v  float64 // U is the cell's left velocity and V is the cell's up velocity
}

type Simulation struct {
	particles []Particle
	grid      []cell
}

func NewSimulation() *Simulation {
	s := Simulation{}
	s.particles = make([]Particle, 0)
	s.addRandomParticles(100)

	cellsW := int(math.Floor(Width/gridSize)) + 1
	cellsH := int(math.Floor(Height/gridSize)) + 1
	totalCells := cellsW * cellsH

	s.grid = make([]cell, totalCells)
	for j := range cellsH {
		for i := range cellsW {
			c := cell{
				// cellType: Air,
				coord: [2]int{i, j},
				u:     0,
				v:     0,
			}

			s.grid[j*cellsW+i] = c
		}
	}

	return &s
}

func (s *Simulation) addRandomParticles(count int) {
	for range count {
		s.particles = append(s.particles, Particle{
			pos: [2]float64{rand.Float64() * Width, rand.Float64() * Height},
			vel: [2]float64{0, 0},
		})
	}
}

func (s *Simulation) Simulate(dt float64) {
	sdt := dt / float64(numSubSteps)

	for range numSubSteps {
		s.integrateParticles(sdt)
	}

}

func (s Simulation) GetParticles() []Particle {
	return s.particles
}

func (s *Simulation) integrateParticles(dt float64) {
	for i := range s.particles {
		s.particles[i].vel[1] += dt * -gravity

		s.particles[i].pos[0] += s.particles[i].vel[0] * dt
		s.particles[i].pos[1] += s.particles[i].vel[1] * dt
	}
}
