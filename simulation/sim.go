package simulation

import (
	"fmt"
	"math"
	"math/rand"
)

const (
	particleCount   int = 8000
	numSubSteps     int = 2
	pressureIters   int = 30
	separationIters int = 5

	flipRatio           float64 = 0.95
	divergenceThreshold float64 = 0.1
	overrelaxation      float64 = 1.9
	driftCompensation   float64 = 1.0
	separationFactor    float64 = 1.0
	separateParticles   bool    = true

	Width      float64 = 4   //m
	Height     float64 = 3   //m
	resolution float64 = 100 // total cells vertically
	GridSize   float64 = Height / resolution

	Radius        float64 = 0.010 //m
	gravity       float64 = -9.81 //ms^-2
	mouseStrength float64 = 5.0   //kgm^3s^-2

	deterministic bool = true
)

var (
	cellsW = int(math.Floor(Width/GridSize)) + 1
	cellsH = int(math.Floor(Height/GridSize)) + 1
)

func init() {
	if Radius >= GridSize {
		fmt.Println("Computation breaks when particles are bigger than gridCells")
	}
}

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

type Cell struct {
	cellType                   CellType
	coord                      [2]int
	u, v                       float64 // U is the cell's left velocity and V is the cell's up velocity
	totalWeightU, totalWeightV float64
	prevU, prevV               float64
	canContainFluid            bool
	particleDensity            float64
}

func (c Cell) GetPos() (x, y int) {
	return c.coord[0], c.coord[1]
}

func (c Cell) Solid() bool {
	return !c.canContainFluid
}

type Simulation struct {
	particles           []Particle
	grid                []Cell
	particleRestDensity float64

	cellAccumulatedParticles []int
	particleLookup           []int
}

func NewSimulation() *Simulation {
	s := Simulation{
		particles:           make([]Particle, 0),
		particleRestDensity: 0.0,
	}
	s.addRandomParticles(particleCount)
	s.particleLookup = make([]int, len(s.particles))

	totalCells := cellsW * cellsH

	s.cellAccumulatedParticles = make([]int, totalCells+1)

	s.grid = make([]Cell, totalCells)
	for j := range cellsH {
		for i := range cellsW {
			c := Cell{
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
	if deterministic {
		r := rand.New(rand.NewSource(2))
		for range count {
			s.particles = append(s.particles, Particle{
				pos: [2]float64{1 + r.Float64()*2, GridSize + r.Float64()*1},
				// pos: [2]float64{rand.Float64() * Width, rand.Float64() * Height},
				vel: [2]float64{0, 0},
			})
		}
	} else {
		for range count {
			s.particles = append(s.particles, Particle{
				pos: [2]float64{1 + rand.Float64()*2, GridSize + rand.Float64()*1},
				// pos: [2]float64{rand.Float64() * Width, rand.Float64() * Height},
				vel: [2]float64{0, 0},
			})
		}

	}
}

func (s *Simulation) Simulate(dt float64, mouseX float64, mouseY float64) {
	sdt := dt / float64(numSubSteps)

	for range numSubSteps {
		s.initialise()
		s.integrateParticles(sdt, mouseX, mouseY)
		if separateParticles {
			s.pushParticlesApart()
		}
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
func (s Simulation) GetGrid() []Cell {
	return s.grid
}

func (s *Simulation) initialise() {
	for i := range s.grid {
		s.grid[i].prevU = s.grid[i].u
		s.grid[i].prevV = s.grid[i].v

		s.grid[i].u = 0
		s.grid[i].v = 0
		s.grid[i].totalWeightU = 0
		s.grid[i].totalWeightV = 0

		if s.grid[i].canContainFluid {
			s.grid[i].cellType = Air
		} else {
			s.grid[i].cellType = Solid
		}

		s.grid[i].particleDensity = 0
	}

}

func (s *Simulation) integrateParticles(dt float64, mouseX float64, mouseY float64) {
	for i := range s.particles {
		s.particles[i].vel[1] += dt * -gravity

		if mouseX >= 0 && mouseY >= 0 {
			dx := s.particles[i].pos[0] - mouseX
			dy := s.particles[i].pos[1] - mouseY
			mouseDist := math.Sqrt(dx*dx + dy*dy)
			mouseDist = math.Max(mouseDist, 0.1)

			mouseForce := mouseStrength / (mouseDist * mouseDist * mouseDist)

			s.particles[i].vel[0] += dt * dx * mouseForce
			s.particles[i].vel[1] += dt * dy * mouseForce
		}

		s.particles[i].pos[0] += s.particles[i].vel[0] * dt
		s.particles[i].pos[1] += s.particles[i].vel[1] * dt
	}
}

func (s *Simulation) handleWallCollisions() {
	minX := GridSize + Radius
	maxX := float64(cellsW-1)*GridSize - Radius
	minY := GridSize + Radius
	maxY := float64(cellsH-1)*GridSize - Radius

	for i := range s.particles {
		x, y := s.particles[i].GetPos()

		if x < minX {
			s.particles[i].pos[0] = minX
			s.particles[i].vel[0] = 0
		} else if x > maxX {
			s.particles[i].pos[0] = maxX
			s.particles[i].vel[0] = 0
		}

		if y < minY {
			s.particles[i].pos[1] = minY
			s.particles[i].vel[1] = 0
		} else if y > maxY {
			s.particles[i].pos[1] = maxY
			s.particles[i].vel[1] = 0
		}
	}
}

func (s *Simulation) transferVelocityToGrid() {
	for i := range s.particles {
		cell := s.particleToCell(s.particles[i])
		if cell.cellType == Air {
			s.grid[cell.coord[1]*cellsW+cell.coord[0]].cellType = Water
		}

		c := s.particleToCell(s.particles[i])
		dx := s.particles[i].pos[0] - float64(c.coord[0])*GridSize
		dy := s.particles[i].pos[1] - float64(c.coord[1])*GridSize

		// u neighbours
		rightNeighbour := c.coord[1]*cellsW + c.coord[0] + 1
		var verticalNeighbour int
		var verticalRightNeighbour int
		inUpper := dy < GridSize/2
		if inUpper {
			targetY := max(c.coord[1]-1, 0)

			verticalNeighbour = targetY*cellsW + c.coord[0]
			verticalRightNeighbour = targetY*cellsW + c.coord[0] + 1
		} else {
			targetY := min(c.coord[1]+1, cellsH-1)

			verticalNeighbour = targetY*cellsW + c.coord[0]
			verticalRightNeighbour = targetY*cellsW + c.coord[0] + 1
		}

		// v neighbours
		belowNeighbour := (c.coord[1]+1)*cellsW + c.coord[0]
		var horizontalNeighbour int
		var horizontalBelowNeighbour int
		inLeft := dx < GridSize/2
		if inLeft {
			targetX := max(c.coord[0]-1, 0)

			horizontalNeighbour = c.coord[1]*cellsW + targetX
			horizontalBelowNeighbour = (c.coord[1]+1)*cellsW + targetX
		} else {
			targetX := min(c.coord[0]+1, cellsW-1)

			horizontalNeighbour = c.coord[1]*cellsW + targetX
			horizontalBelowNeighbour = (c.coord[1]+1)*cellsW + targetX
		}

		var c1u, c2u, c3u, c4u int
		var c1v, c2v, c3v, c4v int
		var tyu, txv float64
		txu := dx / GridSize
		tyv := dy / GridSize

		// orient u
		if inUpper {
			c1u = c.coord[1]*cellsW + c.coord[0]
			c2u = rightNeighbour
			c3u = verticalRightNeighbour
			c4u = verticalNeighbour

			tyu = (dy / GridSize) + 0.5
		} else {
			c1u = verticalNeighbour
			c2u = verticalRightNeighbour
			c3u = rightNeighbour
			c4u = c.coord[1]*cellsW + c.coord[0]

			tyu = (dy / GridSize) - 0.5
		}
		// orient v
		if inLeft {
			c1v = horizontalBelowNeighbour
			c2v = belowNeighbour
			c3v = c.coord[1]*cellsW + c.coord[0]
			c4v = horizontalNeighbour

			txv = (dx / GridSize) + 0.5
		} else {
			c1v = belowNeighbour
			c2v = horizontalBelowNeighbour
			c3v = horizontalNeighbour
			c4v = c.coord[1]*cellsW + c.coord[0]

			txv = (dx / GridSize) - 0.5
		}

		w1u := (1 - txu) * (tyu)
		w2u := (txu) * (tyu)
		w3u := (txu) * (1 - tyu)
		w4u := (1 - txu) * (1 - tyu)

		w1v := (1 - txv) * (tyv)
		w2v := (txv) * (tyv)
		w3v := (txv) * (1 - tyv)
		w4v := (1 - txv) * (1 - tyv)

		pVelX := s.particles[i].vel[0]
		pVelY := s.particles[i].vel[1]

		// bilinearly add velocity to surrounding edges
		s.grid[c1u].u += w1u * pVelX
		s.grid[c2u].u += w2u * pVelX
		s.grid[c3u].u += w3u * pVelX
		s.grid[c4u].u += w4u * pVelX

		s.grid[c1v].v += w1v * pVelY
		s.grid[c2v].v += w2v * pVelY
		s.grid[c3v].v += w3v * pVelY
		s.grid[c4v].v += w4v * pVelY

		// store total weight on the particles
		s.grid[c1u].totalWeightU += w1u
		s.grid[c2u].totalWeightU += w2u
		s.grid[c3u].totalWeightU += w3u
		s.grid[c4u].totalWeightU += w4u

		s.grid[c1v].totalWeightV += w1v
		s.grid[c2v].totalWeightV += w2v
		s.grid[c3v].totalWeightV += w3v
		s.grid[c4v].totalWeightV += w4v
	}
	for i := range s.grid {
		// revert solid edges to previous velocity
		// otherwise normalilse velocity by totalWeight

		if !s.grid[i].canContainFluid || (s.grid[i].coord[0] > 0 && !s.grid[i-1].canContainFluid) {
			s.grid[i].u = s.grid[i].prevU

		} else if s.grid[i].totalWeightU > 0 {
			s.grid[i].u /= s.grid[i].totalWeightU

		}
		if !s.grid[i].canContainFluid || (s.grid[i].coord[1] > 0 && !s.grid[(s.grid[i].coord[1]-1)*cellsW+s.grid[i].coord[0]].canContainFluid) {
			s.grid[i].v = s.grid[i].prevV

		} else if s.grid[i].totalWeightV > 0 {
			s.grid[i].v /= s.grid[i].totalWeightV

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

				divergence := s.grid[right].u - s.grid[center].u + s.grid[down].v - s.grid[center].v
				if divergence*divergence < divergenceThreshold {
					continue
				}

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

	for i := range s.particles {
		c := s.particleToCell(s.particles[i])
		dx := s.particles[i].pos[0] - float64(c.coord[0])*GridSize
		dy := s.particles[i].pos[1] - float64(c.coord[1])*GridSize

		rightNeighbour := c.coord[1]*cellsW + c.coord[0] + 1
		var verticalNeighbour int
		var verticalRightNeighbour int
		inUpper := dy < GridSize/2
		if inUpper {
			targetY := max(c.coord[1]-1, 0)

			verticalNeighbour = targetY*cellsW + c.coord[0]
			verticalRightNeighbour = targetY*cellsW + c.coord[0] + 1
		} else {
			targetY := min(c.coord[1]+1, cellsH-1)

			verticalNeighbour = targetY*cellsW + c.coord[0]
			verticalRightNeighbour = targetY*cellsW + c.coord[0] + 1
		}

		belowNeighbour := (c.coord[1]+1)*cellsW + c.coord[0]
		var horizontalNeighbour int
		var horizontalBelowNeighbour int
		inLeft := dx < GridSize/2
		if inLeft {
			targetX := max(c.coord[0]-1, 0)

			horizontalNeighbour = c.coord[1]*cellsW + targetX
			horizontalBelowNeighbour = (c.coord[1]+1)*cellsW + targetX
		} else {
			targetX := min(c.coord[0]+1, cellsW-1)

			horizontalNeighbour = c.coord[1]*cellsW + targetX
			horizontalBelowNeighbour = (c.coord[1]+1)*cellsW + targetX
		}

		var c1u, c2u, c3u, c4u int
		var c1v, c2v, c3v, c4v int
		var tyu, txv float64
		txu := dx / GridSize
		tyv := dy / GridSize

		if inUpper {
			c1u = c.coord[1]*cellsW + c.coord[0]
			c2u = rightNeighbour
			c3u = verticalRightNeighbour
			c4u = verticalNeighbour

			tyu = tyv + 0.5
		} else {
			c1u = verticalNeighbour
			c2u = verticalRightNeighbour
			c3u = rightNeighbour
			c4u = c.coord[1]*cellsW + c.coord[0]

			tyu = tyv - 0.5
		}

		if inLeft {
			c1v = horizontalBelowNeighbour
			c2v = belowNeighbour
			c3v = c.coord[1]*cellsW + c.coord[0]
			c4v = horizontalNeighbour

			txv = txu + 0.5
		} else {
			c1v = belowNeighbour
			c2v = horizontalBelowNeighbour
			c3v = horizontalNeighbour
			c4v = c.coord[1]*cellsW + c.coord[0]

			txv = txu - 0.5
		}

		w1u := (1 - txu) * (tyu)
		w2u := (txu) * (tyu)
		w3u := (txu) * (1 - tyu)
		w4u := (1 - txu) * (1 - tyu)

		w1v := (1 - txv) * (tyv)
		w2v := (txv) * (tyv)
		w3v := (txv) * (1 - tyv)
		w4v := (1 - txv) * (1 - tyv)

		pVelX := s.particles[i].vel[0]
		pVelY := s.particles[i].vel[1]

		valid1u, valid2u, valid3u, valid4u := 0.0, 0.0, 0.0, 0.0
		if s.grid[c1u].cellType != Air || (s.grid[c1u].coord[0] > 0 && s.grid[c1u-1].cellType != Air) {
			valid1u = 1
		}
		if s.grid[c2u].cellType != Air || (s.grid[c2u].coord[0] > 0 && s.grid[c2u-1].cellType != Air) {
			valid2u = 1
		}
		if s.grid[c3u].cellType != Air || (s.grid[c3u].coord[0] > 0 && s.grid[c3u-1].cellType != Air) {
			valid3u = 1
		}
		if s.grid[c4u].cellType != Air || (s.grid[c4u].coord[0] > 0 && s.grid[c4u-1].cellType != Air) {
			valid4u = 1
		}
		if wu := valid1u*w1u + valid2u*w2u + valid3u*w3u + valid4u*w4u; wu > 0 {
			picV := (valid1u*w1u*s.grid[c1u].u +
				valid2u*w2u*s.grid[c2u].u +
				valid3u*w3u*s.grid[c3u].u +
				valid4u*w4u*s.grid[c4u].u) / wu
			corr := (valid1u*w1u*(s.grid[c1u].u-s.grid[c1u].prevU) +
				valid2u*w2u*(s.grid[c2u].u-s.grid[c2u].prevU) +
				valid3u*w3u*(s.grid[c3u].u-s.grid[c3u].prevU) +
				valid4u*w4u*(s.grid[c4u].u-s.grid[c4u].prevU)) / wu
			flipV := pVelX + corr

			s.particles[i].vel[0] = (1.0-flipRatio)*picV + flipRatio*flipV
		}

		valid1v, valid2v, valid3v, valid4v := 0.0, 0.0, 0.0, 0.0
		if s.grid[c1v].cellType != Air || (s.grid[c1v].coord[1] > 0 && s.grid[c1v-cellsW].cellType != Air) {
			valid1v = 1
		}
		if s.grid[c2v].cellType != Air || (s.grid[c2v].coord[1] > 0 && s.grid[c2v-cellsW].cellType != Air) {
			valid2v = 1
		}
		if s.grid[c3v].cellType != Air || (s.grid[c3v].coord[1] > 0 && s.grid[c3v-cellsW].cellType != Air) {
			valid3v = 1
		}
		if s.grid[c4v].cellType != Air || (s.grid[c4v].coord[1] > 0 && s.grid[c4v-cellsW].cellType != Air) {
			valid4v = 1
		}
		if wv := valid1v*w1v + valid2v*w2v + valid3v*w3v + valid4v*w4v; wv > 0 {
			picV := (valid1v*w1v*s.grid[c1v].v +
				valid2v*w2v*s.grid[c2v].v +
				valid3v*w3v*s.grid[c3v].v +
				valid4v*w4v*s.grid[c4v].v) / wv
			corr := (valid1v*w1v*(s.grid[c1v].v-s.grid[c1v].prevV) +
				valid2v*w2v*(s.grid[c2v].v-s.grid[c2v].prevV) +
				valid3v*w3v*(s.grid[c3v].v-s.grid[c3v].prevV) +
				valid4v*w4v*(s.grid[c4v].v-s.grid[c4v].prevV)) / wv
			flipV := pVelY + corr

			s.particles[i].vel[1] = (1.0-flipRatio)*picV + flipRatio*flipV
		}
	}
}

func (s *Simulation) updateParticleDensity() {
	for i := range s.particles {

		shiftedX := math.Max(0, math.Min(s.particles[i].pos[0]-(GridSize*0.5), float64(cellsW-2)*GridSize))
		shiftedY := math.Max(0, math.Min(s.particles[i].pos[1]-(GridSize*0.5), float64(cellsH-2)*GridSize))

		cx := int(math.Floor(shiftedX / GridSize))
		cy := int(math.Floor(shiftedY / GridSize))

		tx := (shiftedX - float64(cx)*GridSize) / GridSize
		ty := (shiftedY - float64(cy)*GridSize) / GridSize
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

func (s *Simulation) pushParticlesApart() {
	s.buildSpacialHash()

	for range separationIters {
		for i := range s.particles {
			cellX := int(s.particles[i].pos[0] / GridSize)
			cellY := int(s.particles[i].pos[1] / GridSize)

			for neighbourY := cellY - 1; neighbourY <= cellY+1; neighbourY++ {
				for neighbourX := cellX - 1; neighbourX <= cellX+1; neighbourX++ {
					if neighbourX < 0 || neighbourX >= cellsW || neighbourY < 0 || neighbourY >= cellsH {
						continue
					}
					id := neighbourY*cellsW + neighbourX

					firstParticle := s.cellAccumulatedParticles[id]  //number of particles before this cell
					lastParticle := s.cellAccumulatedParticles[id+1] //number of particles before next cell

					for otherParticle := firstParticle; otherParticle < lastParticle; otherParticle++ {
						if i == s.particleLookup[otherParticle] {
							continue
						}

						dx := s.particles[i].pos[0] - s.particles[s.particleLookup[otherParticle]].pos[0]
						dy := s.particles[i].pos[1] - s.particles[s.particleLookup[otherParticle]].pos[1]
						distSquared := dx*dx + dy*dy

						if distSquared == 0 {
							continue
						}

						if distSquared < 4*Radius*Radius {
							dist := math.Sqrt(distSquared)
							overlap := 2*Radius - dist

							normalisedX := dx / dist
							normalisedY := dy / dist

							pushAmount := overlap * 0.5 * separationFactor

							s.particles[i].pos[0] += normalisedX * pushAmount
							s.particles[i].pos[1] += normalisedY * pushAmount

							s.particles[s.particleLookup[otherParticle]].pos[0] -= normalisedX * pushAmount
							s.particles[s.particleLookup[otherParticle]].pos[1] -= normalisedY * pushAmount
						}
					}
				}
			}

		}
	}
}

func (s *Simulation) buildSpacialHash() {
	for i := range s.cellAccumulatedParticles {
		s.cellAccumulatedParticles[i] = 0
	}
	for i := range s.particles {
		cx := int(s.particles[i].pos[0] / GridSize)
		cy := int(s.particles[i].pos[1] / GridSize)

		if cx < 0 || cx >= cellsW || cy < 0 || cy >= cellsH {
			fmt.Println(cellsW, cellsH, "Out of bounds", cx, cy) //TODO TEMPORARY FIX FOR THIS OOB ERROR
			cx = max(0, min(cx, cellsW-1))
			cy = max(0, min(cy, cellsH-1))
			fmt.Println("changed to:", cx, cy)
		}

		cellId := cy*cellsW + cx

		s.cellAccumulatedParticles[cellId]++ //how many are in each cell
	}

	currentSum := 0
	for i := range s.grid {
		count := s.cellAccumulatedParticles[i]
		s.cellAccumulatedParticles[i] = currentSum // cumulative "frequency"
		currentSum += count
	}
	s.cellAccumulatedParticles[len(s.grid)] = currentSum

	for i := range s.particles {
		cx := int(s.particles[i].pos[0] / GridSize)
		cy := int(s.particles[i].pos[1] / GridSize)

		if cx < 0 || cx >= cellsW || cy < 0 || cy >= cellsH {
			fmt.Println(cellsW, cellsH, "Also Out of bounds", cx, cy)
			cx = max(0, min(cx, cellsW-1))
			cy = max(0, min(cy, cellsH-1))
			fmt.Println("changed to:", cx, cy)
		}

		cellId := cy*cellsW + cx

		freeID := s.cellAccumulatedParticles[cellId]
		s.particleLookup[freeID] = i //load particleLookup with particle ids

		s.cellAccumulatedParticles[cellId]++
	}

	for i := len(s.grid) - 1; i > 0; i-- {
		s.cellAccumulatedParticles[i] = s.cellAccumulatedParticles[i-1] // restore accumulated count
	}
	s.cellAccumulatedParticles[0] = 0
}

func (s *Simulation) particleToCell(p Particle) Cell {
	x, y := p.GetPos()

	xCell := int(math.Floor(x / GridSize))
	yCell := int(math.Floor(y / GridSize))

	return s.grid[yCell*cellsW+xCell]
}
