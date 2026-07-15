package simulation

import (
	"math"
	"math/rand"
)

const (
	numSubSteps       int     = 3
	pressureIters     int     = 30
	flipRatio         float64 = 0.9
	overrelaxation    float64 = 1.9
	driftCompensation float64 = 1.0
	particleCount     int     = 2000
	gravity           float64 = -9.81 //ms^-2

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
	particleDensity float64
}

type Simulation struct {
	particles           []Particle
	grid                []cell
	particleRestDensity float64
}

func NewSimulation() *Simulation {
	s := Simulation{}
	s.particles = make([]Particle, 0)
	s.addRandomParticles(particleCount)

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
			if i == 0 || i == cellsW-1 || j == cellsH-1 || j == 0 {
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
			pos: [2]float64{1 + rand.Float64()*2, rand.Float64() * 1},
			// pos: [2]float64{rand.Float64() * Width, rand.Float64() * Height},
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
		s.updateParticleDensity()
		s.solveIncompressibility()
		s.transferVelocityToParticles()
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
			s.grid[cell.coord[1]*cellsW+cell.coord[0]].cellType = Water
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

func (s *Simulation) solveIncompressibility() {
	for i := range s.grid {
		s.grid[i].prevU = s.grid[i].u
		s.grid[i].prevV = s.grid[i].v
	}

	for range pressureIters {
		for j := 1; j < cellsH-1; j++ {
			for i := 1; i < cellsW-1; i++ {
				if s.grid[j*cellsW+i].cellType != Water {
					continue
				}

				center := j*cellsW + i
				up := (j-1)*cellsW + i
				down := (j+1)*cellsW + i
				left := j*cellsW + i - 1
				right := j*cellsW + i + 1

				openNeighbours := 0
				if s.grid[up].canContainFluid {
					openNeighbours++
				}
				if s.grid[down].canContainFluid {
					openNeighbours++
				}
				if s.grid[left].canContainFluid {
					openNeighbours++
				}
				if s.grid[right].canContainFluid {
					openNeighbours++
				}
				if openNeighbours == 0 {
					continue
				}

				divergence := s.grid[right].u - s.grid[center].u + s.grid[down].v - s.grid[center].v

				if s.particleRestDensity > 0 {
					compression := s.grid[center].particleDensity - s.particleRestDensity
					if compression > 0 {
						divergence -= compression * driftCompensation
					}
				}

				p := divergence / float64(openNeighbours)

				p *= overrelaxation

				if s.grid[right].canContainFluid {
					s.grid[right].u -= p
				}
				if s.grid[left].canContainFluid {
					s.grid[center].u += p
				}
				if s.grid[up].canContainFluid {
					s.grid[center].v += p
				}
				if s.grid[down].canContainFluid {
					s.grid[down].v -= p
				}
			}
		}
	}
}

func (s *Simulation) transferVelocityToParticles() {

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

		valid1, valid2, valid3, valid4 := 0.0, 0.0, 0.0, 0.0
		if s.grid[c1].cellType != Air || (c1 > 0 && s.grid[c1-1].cellType != Air) {
			valid1 = 1
		}
		if s.grid[c2].cellType != Air || (c2 > 0 && s.grid[c2-1].cellType != Air) {
			valid2 = 1
		}
		if s.grid[c3].cellType != Air || (c3 > 0 && s.grid[c3-1].cellType != Air) {
			valid3 = 1
		}
		if s.grid[c4].cellType != Air || (c4 > 0 && s.grid[c4-1].cellType != Air) {
			valid4 = 1
		}
		w := valid1*w1 + valid2*w2 + valid3*w3 + valid4*w4

		if w > 0 {
			picV := (valid1*w1*s.grid[c1].u +
				valid2*w2*s.grid[c2].u +
				valid3*w3*s.grid[c3].u +
				valid4*w4*s.grid[c4].u) / w
			corr := (valid1*w1*(s.grid[c1].u-s.grid[c1].prevU) +
				valid2*w2*(s.grid[c2].u-s.grid[c2].prevU) +
				valid3*w3*(s.grid[c3].u-s.grid[c3].prevU) +
				valid4*w4*(s.grid[c4].u-s.grid[c4].prevU)) / w
			flipV := pVelX + corr

			s.particles[i].vel[0] = (1.0-flipRatio)*picV + flipRatio*flipV
		}
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

		valid1, valid2, valid3, valid4 := 0.0, 0.0, 0.0, 0.0
		if s.grid[c1].cellType != Air || (c1 >= cellsW && s.grid[c1-cellsW].cellType != Air) {
			valid1 = 1
		}
		if s.grid[c2].cellType != Air || (c2 >= cellsW && s.grid[c2-cellsW].cellType != Air) {
			valid2 = 1
		}
		if s.grid[c3].cellType != Air || (c3 >= cellsW && s.grid[c3-cellsW].cellType != Air) {
			valid3 = 1
		}
		if s.grid[c4].cellType != Air || (c4 >= cellsW && s.grid[c4-cellsW].cellType != Air) {
			valid4 = 1
		}
		w := valid1*w1 + valid2*w2 + valid3*w3 + valid4*w4

		if w > 0 {
			picV := (valid1*w1*s.grid[c1].v +
				valid2*w2*s.grid[c2].v +
				valid3*w3*s.grid[c3].v +
				valid4*w4*s.grid[c4].v) / w
			corr := (valid1*w1*(s.grid[c1].v-s.grid[c1].prevV) +
				valid2*w2*(s.grid[c2].v-s.grid[c2].prevV) +
				valid3*w3*(s.grid[c3].v-s.grid[c3].prevV) +
				valid4*w4*(s.grid[c4].v-s.grid[c4].prevV)) / w
			flipV := pVelY + corr

			s.particles[i].vel[1] = (1.0-flipRatio)*picV + flipRatio*flipV
		}
	}
}

func (s *Simulation) updateParticleDensity() {
	for i := range s.grid {
		s.grid[i].particleDensity = 0
	}
	for i := range s.particles {

		shiftedX := math.Max(0, math.Min(s.particles[i].pos[0]-(gridSize*0.5), float64(cellsW-2)*gridSize))
		shiftedY := math.Max(0, math.Min(s.particles[i].pos[1]-(gridSize*0.5), float64(cellsH-2)*gridSize))

		cx := int(math.Floor(shiftedX / gridSize))
		cy := int(math.Floor(shiftedY / gridSize))

		tx := (shiftedX - float64(cx)*gridSize) / gridSize
		ty := (shiftedY - float64(cy)*gridSize) / gridSize
		sx := 1.0 - tx
		sy := 1.0 - ty

		w1 := sx * ty
		w2 := tx * ty
		w3 := tx * sy
		w4 := sx * sy

		c1 := (cy+1)*cellsW + cx
		c2 := (cy+1)*cellsW + (cx + 1)
		c3 := cy*cellsW + (cx + 1)
		c4 := cy*cellsW + cx

		s.grid[c1].particleDensity += w1
		s.grid[c2].particleDensity += w2
		s.grid[c3].particleDensity += w3
		s.grid[c4].particleDensity += w4
	}
	if s.particleRestDensity == 0 {
		sum := 0.0
		fluidCellCount := 0.0

		for i := range s.grid {
			if s.grid[i].cellType == Water {
				sum += s.grid[i].particleDensity
				fluidCellCount++
			}
		}
		if fluidCellCount > 0 {
			s.particleRestDensity = sum / fluidCellCount
		}
	}
}

func (s *Simulation) particleToCell(p Particle) cell {
	x, y := p.GetPos()

	xCell := int(math.Floor(x / gridSize))
	yCell := int(math.Floor(y / gridSize))

	return s.grid[yCell*cellsW+xCell]
}
