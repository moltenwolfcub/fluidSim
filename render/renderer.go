package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/moltenwolfcub/fluidSim/simulation"
)

const (
	WindowWidth  = 1600
	WindowHeight = 900
	TPS          = 60
)

type Renderer struct {
	sim *simulation.Simulation
}

func NewRenderer(sim *simulation.Simulation) *Renderer {
	r := Renderer{
		sim: sim,
	}

	return &r
}

func (g *Renderer) Run() error {
	ebiten.SetWindowSize(960, 540)
	ebiten.SetWindowTitle("Fluid Simulation")
	ebiten.SetTPS(TPS)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(g)
}

func (g *Renderer) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return WindowWidth, WindowHeight
}

func (g *Renderer) Update() error {
	return nil
}

func (g *Renderer) Draw(screen *ebiten.Image) {
	for _, p := range g.sim.GetParticles() {
		g.drawParticle(screen, p)
	}
}

func (g *Renderer) drawParticle(screen *ebiten.Image, p simulation.Particle) {
	x, y := SimToRenderCoords(p.GetPos())

	vector.FillCircle(screen, float32(x), float32(y), float32(simulation.Radius), color.RGBA{71, 155, 203, 255}, true)
}

func SimToRenderCoords(xSim, ySim float64) (xRender, yRender float64) {
	xRender = xSim * (WindowWidth / simulation.Width)
	yRender = ySim * (WindowHeight / simulation.Height)
	return
}
