package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/moltenwolfcub/fluidSim/render"
	"github.com/moltenwolfcub/fluidSim/simulation"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	sim := simulation.NewSimulation()
	render := render.NewRenderer(sim)

	render.Run()
}
