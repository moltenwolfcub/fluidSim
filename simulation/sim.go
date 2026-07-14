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

	Radius float64 = 0.025 //m
)

var (
	cellsW = int(math.Floor(Width/gridSize)) + 1
	cellsH = int(math.Floor(Height/gridSize)) + 1
)

type Particle struct {
	pos [2]float64
	vel [2]float64
}

func (p Particle) GetPos() (x, y float64) {
	return p.pos[0], p.pos[1]
}

type CellType int

const (
	Air CellType = iota
	Water
	Solid
)

type cell struct {
	cellType        CellType
	coord           [2]int
	u, v            float64 // U is the cell's left velocity and V is the cell's up velocity
	du, dv          float64
	prevU, prevV    float64
	canContainFluid bool
}

type Simulation struct {
	particles []Particle
	grid      []cell
}

func NewSimulation() *Simulation {
	s := Simulation{}
	s.particles = make([]Particle, 0)
	s.addRandomParticles(100)

	totalCells := cellsW * cellsH

	s.grid = make([]cell, totalCells)
	for j := range cellsH {
		for i := range cellsW {
			c := cell{
				cellType: Air,
				coord:    [2]int{i, j},
				u:        0,
				v:        0,
			}
			c.canContainFluid = true
			if i == 0 || i == cellsW-1 || j == 0 {
				c.canContainFluid = false
				c.cellType = Solid
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
		s.handleWallCollisions()
		s.transferVelocityToGrid()
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

func (s *Simulation) handleWallCollisions() {
	for i := range s.particles {
		x, y := s.particles[i].GetPos()

		if x < Radius {
			s.particles[i].pos[0] = Radius
			s.particles[i].vel[0] = 0
		} else if x > Width-Radius {
			s.particles[i].pos[0] = Width - Radius
			s.particles[i].vel[0] = 0
		}

		if y < Radius {
			s.particles[i].pos[1] = Radius
			s.particles[i].vel[1] = 0
		} else if y > Height-Radius {
			s.particles[i].pos[1] = Height - Radius
			s.particles[i].vel[1] = 0
		}
	}
}

func (s *Simulation) transferVelocityToGrid() {
	for i := range s.grid {
		s.grid[i].prevU = s.grid[i].u
		s.grid[i].prevV = s.grid[i].v

		s.grid[i].u = 0
		s.grid[i].v = 0
		s.grid[i].du = 0
		s.grid[i].dv = 0

		if s.grid[i].canContainFluid {
			s.grid[i].cellType = Air
		} else {
			s.grid[i].cellType = Solid
		}
	}
	for i := range s.particles {
		cell := s.particleToCell(s.particles[i])
		if cell.cellType == Air {
			cell.cellType = Water
		}
	}

	// x-component
	for i := range s.particles {
		c := s.particleToCell(s.particles[i])
		dx := s.particles[i].pos[0] - float64(c.coord[0])*gridSize
		dy := s.particles[i].pos[1] - float64(c.coord[1])*gridSize

		rightNeighbour := c.coord[1]*cellsW + c.coord[0] + 1
		var verticalNeighbour int
		var verticalRightNeighbour int
		inUpper := dy < gridSize/2
		if inUpper {
			targetY := max(c.coord[1]-1, 0)

			verticalNeighbour = targetY*cellsW + c.coord[0]
			verticalRightNeighbour = targetY*cellsW + c.coord[0] + 1
		} else {
			targetY := min(c.coord[1]+1, cellsH-1)

			verticalNeighbour = targetY*cellsW + c.coord[0]
			verticalRightNeighbour = targetY*cellsW + c.coord[0] + 1
		}

		var c1, c2, c3, c4 int
		var ty float64
		tx := dx / gridSize

		if inUpper {
			c1 = c.coord[1]*cellsW + c.coord[0]
			c2 = rightNeighbour
			c3 = verticalRightNeighbour
			c4 = verticalNeighbour

			ty = (dy / gridSize) + 0.5
		} else {
			c1 = verticalNeighbour
			c2 = verticalRightNeighbour
			c3 = rightNeighbour
			c4 = c.coord[1]*cellsW + c.coord[0]

			ty = (dy / gridSize) - 0.5
		}

		w1 := (1 - tx) * (ty)
		w2 := (tx) * (ty)
		w3 := (tx) * (1 - ty)
		w4 := (1 - tx) * (1 - ty)

		pVelX := s.particles[i].vel[0]

		s.grid[c1].u += w1 * pVelX
		s.grid[c2].u += w2 * pVelX
		s.grid[c3].u += w3 * pVelX
		s.grid[c4].u += w4 * pVelX

		s.grid[c1].du += w1
		s.grid[c2].du += w2
		s.grid[c3].du += w3
		s.grid[c4].du += w4
	}
	// y-component
	for i := range s.particles {
		c := s.particleToCell(s.particles[i])
		dx := s.particles[i].pos[0] - float64(c.coord[0])*gridSize
		dy := s.particles[i].pos[1] - float64(c.coord[1])*gridSize

		belowNeighbour := (c.coord[1]+1)*cellsW + c.coord[0]
		var horizontalNeighbour int
		var horizontalBelowNeighbour int
		inLeft := dx < gridSize/2
		if inLeft {
			targetX := max(c.coord[0]-1, 0)

			horizontalNeighbour = c.coord[1]*cellsW + targetX
			horizontalBelowNeighbour = (c.coord[1]+1)*cellsW + targetX
		} else {
			targetX := min(c.coord[0]+1, cellsW-1)

			horizontalNeighbour = c.coord[1]*cellsW + targetX
			horizontalBelowNeighbour = (c.coord[1]+1)*cellsW + targetX
		}

		var c1, c2, c3, c4 int
		var tx float64
		ty := dy / gridSize

		if inLeft {
			c1 = horizontalBelowNeighbour
			c2 = belowNeighbour
			c3 = c.coord[1]*cellsW + c.coord[0]
			c4 = horizontalNeighbour

			tx = (dx / gridSize) + 0.5
		} else {
			c1 = belowNeighbour
			c2 = horizontalBelowNeighbour
			c3 = horizontalNeighbour
			c4 = c.coord[1]*cellsW + c.coord[0]

			tx = (dx / gridSize) - 0.5
		}

		w1 := (1 - tx) * (ty)
		w2 := (tx) * (ty)
		w3 := (tx) * (1 - ty)
		w4 := (1 - tx) * (1 - ty)

		pVelY := s.particles[i].vel[1]

		s.grid[c1].v += w1 * pVelY
		s.grid[c2].v += w2 * pVelY
		s.grid[c3].v += w3 * pVelY
		s.grid[c4].v += w4 * pVelY

		s.grid[c1].dv += w1
		s.grid[c2].dv += w2
		s.grid[c3].dv += w3
		s.grid[c4].dv += w4
	}
	for i := range s.grid {
		if s.grid[i].du > 0 {
			s.grid[i].u /= s.grid[i].du
		}
		if s.grid[i].dv > 0 {
			s.grid[i].v /= s.grid[i].dv
		}
	}
	for i := range s.grid { //revert solid edges to previous velocity
		if !s.grid[i].canContainFluid || (s.grid[i].coord[0] > 0 && !s.grid[i-1].canContainFluid) {
			s.grid[i].u = s.grid[i].prevU
		}
		if !s.grid[i].canContainFluid || (s.grid[i].coord[1] > 0 && !s.grid[(s.grid[i].coord[1]-1)*cellsW+s.grid[i].coord[0]].canContainFluid) {
			s.grid[i].v = s.grid[i].prevV
		}
	}
}

func (s *Simulation) particleToCell(p Particle) cell {
	x, y := p.GetPos()

	xCell := int(math.Floor(x / gridSize))
	yCell := int(math.Floor(y / gridSize))

	return s.grid[yCell*cellsW+xCell]
}
