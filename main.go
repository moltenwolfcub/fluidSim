package main

import (
	"github.com/moltenwolfcub/fluidSim/render"
	"github.com/moltenwolfcub/fluidSim/simulation"
)

func main() {
	sim := simulation.NewSimulation()
	render := render.NewRenderer(sim)

	render.Run()
}
