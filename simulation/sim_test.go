package simulation

import (
	"testing"
)

func setTestingParameters() {
	deterministic = true
	particleCount = 10_000
	numSubSteps = 3
	pressureIters = 50
	separationIters = 2

	flipRatio = 0.95
	divergenceThreshold = 0.1
	overrelaxation = 1.9
	driftCompensation = 1.0
	separationFactor = 1.0
	separateParticles = true

	Width = 4
	Height = 3
	resolution = 100
	GridSize = Height / resolution

	Radius = 0.010
	gravity = -9.81
	mouseStrength = 5.0
}

func BenchmarkSimulate(b *testing.B) {
	setTestingParameters()

	sim := NewSimulation()

	for b.Loop() {

		sim.Simulate(1.0/60.0, -1, -1)
	}
}

func BenchmarkInitialise(b *testing.B) {
	setTestingParameters()

	sim := NewSimulation()

	for b.Loop() {
		sim.initialise()
	}
}

func BenchmarkIntegrateParticles(b *testing.B) {
	setTestingParameters()

	sim := NewSimulation()

	sdt := (1.0 / 60.0) / float64(numSubSteps)

	for b.Loop() {
		sim.integrateParticles(sdt, -1, -1)
	}
}

func BenchmarkPushParticlesApart(b *testing.B) {
	setTestingParameters()

	sim := NewSimulation()

	sdt := (1.0 / 60.0) / float64(numSubSteps)

	for b.Loop() {
		b.StopTimer()

		sim.initialise()
		sim.integrateParticles(sdt, -1, -1)

		b.StartTimer()
		sim.pushParticlesApart()
		b.StopTimer()

		sim.handleWallCollisions()
		sim.transferVelocityToGrid()
		sim.updateParticleDensity()
		sim.solveIncompressibility()
		sim.transferVelocityToParticles()

		b.StartTimer()
	}
}

func BenchmarkHandleWallCollisons(b *testing.B) {
	setTestingParameters()

	sim := NewSimulation()

	sdt := (1.0 / 60.0) / float64(numSubSteps)

	for b.Loop() {
		b.StopTimer()

		sim.initialise()
		sim.integrateParticles(sdt, -1, -1)
		sim.pushParticlesApart()

		b.StartTimer()
		sim.handleWallCollisions()
		b.StopTimer()

		sim.transferVelocityToGrid()
		sim.updateParticleDensity()
		sim.solveIncompressibility()
		sim.transferVelocityToParticles()

		b.StartTimer()
	}
}

func BenchmarkTransferVelocityToGrid(b *testing.B) {
	setTestingParameters()

	sim := NewSimulation()

	sdt := (1.0 / 60.0) / float64(numSubSteps)

	for b.Loop() {
		b.StopTimer()

		sim.initialise()
		sim.integrateParticles(sdt, -1, -1)
		sim.pushParticlesApart()
		sim.handleWallCollisions()

		b.StartTimer()
		sim.transferVelocityToGrid()
		b.StopTimer()

		sim.updateParticleDensity()
		sim.solveIncompressibility()
		sim.transferVelocityToParticles()

		b.StartTimer()
	}
}

func BenchmarkUpdateParticleDensity(b *testing.B) {
	setTestingParameters()

	sim := NewSimulation()

	sdt := (1.0 / 60.0) / float64(numSubSteps)

	for b.Loop() {
		b.StopTimer()

		sim.initialise()
		sim.integrateParticles(sdt, -1, -1)
		sim.pushParticlesApart()
		sim.handleWallCollisions()
		sim.transferVelocityToGrid()

		b.StartTimer()
		sim.updateParticleDensity()
		b.StopTimer()

		sim.solveIncompressibility()
		sim.transferVelocityToParticles()

		b.StartTimer()
	}
}

func BenchmarkSolveIncompressibility(b *testing.B) {
	setTestingParameters()

	sim := NewSimulation()

	sdt := (1.0 / 60.0) / float64(numSubSteps)

	for b.Loop() {
		b.StopTimer()

		sim.initialise()
		sim.integrateParticles(sdt, -1, -1)
		sim.pushParticlesApart()
		sim.handleWallCollisions()
		sim.transferVelocityToGrid()
		sim.updateParticleDensity()

		b.StartTimer()
		sim.solveIncompressibility()
		b.StopTimer()

		sim.transferVelocityToParticles()

		b.StartTimer()
	}
}

func BenchmarkTransferVelocityToParticles(b *testing.B) {
	setTestingParameters()

	sim := NewSimulation()

	sdt := (1.0 / 60.0) / float64(numSubSteps)

	for b.Loop() {
		b.StopTimer()

		sim.initialise()
		sim.integrateParticles(sdt, -1, -1)
		sim.pushParticlesApart()
		sim.handleWallCollisions()
		sim.transferVelocityToGrid()
		sim.updateParticleDensity()
		sim.solveIncompressibility()

		b.StartTimer()
		sim.transferVelocityToParticles()
	}
}
