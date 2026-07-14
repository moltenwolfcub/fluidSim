package simulation

import "math/rand"

const (
	gridSize float64 = 50
	Width    float64 = 1600
	Height   float64 = 900

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

	cellsW := int(Width/gridSize) + 1
	cellsH := int(Height/gridSize) + 1
	totalCells := cellsW * cellsH

	s.grid = make([]cell, totalCells)
	for j := 0; j < cellsH; j++ {
		for i := 0; i < cellsW; i++ {
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

func (s Simulation) GetParticles() []Particle {
	return s.particles
}
