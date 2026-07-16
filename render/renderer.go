package render

import (
	"image/color"
	"math"

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
	sim              *simulation.Simulation
	cachedSolidTiles *ebiten.Image
	cachedParticle   *ebiten.Image
	particleRadius   int
}

func NewRenderer(sim *simulation.Simulation) *Renderer {
	r := Renderer{
		sim: sim,
	}

	r.cachedSolidTiles = ebiten.NewImage(r.Layout(0, 0))
	for _, c := range r.sim.GetGrid() {
		if c.Solid() {
			r.drawCell(r.cachedSolidTiles, c)
		}
	}

	r.particleRadius = int(math.Floor(simulation.Radius * (WindowHeight / simulation.Height)))
	r.cachedParticle = ebiten.NewImage(r.particleRadius*2, r.particleRadius*2)
	vector.FillCircle(r.cachedParticle, float32(r.particleRadius), float32(r.particleRadius), float32(r.particleRadius), color.RGBA{71, 155, 203, 255}, true)

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
	var mx, my float64
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		rawX, rawY := ebiten.CursorPosition()

		mx = float64(rawX) * (simulation.Width / WindowWidth)
		my = float64(rawY) * (simulation.Height / WindowHeight)

	} else {
		mx, my = -1, -1
	}

	g.sim.Simulate(1/float64(TPS), mx, my)
	return nil
}

func (g *Renderer) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{14, 30, 39, 255})
	screen.DrawImage(g.cachedSolidTiles, nil)

	for _, p := range g.sim.GetParticles() {
		g.drawParticle(screen, p)
	}
}

func (g *Renderer) drawCell(screen *ebiten.Image, c simulation.Cell) {
	rawX, rawY := c.GetPos()
	x, y := SimToRenderCoords(float64(rawX)*simulation.GridSize, float64(rawY)*simulation.GridSize)

	vector.FillRect(screen, float32(x), float32(y), float32(simulation.GridSize*(WindowWidth/simulation.Width)), float32(simulation.GridSize*(WindowHeight/simulation.Height)), color.RGBA{100, 100, 100, 255}, true)
}

func (g *Renderer) drawParticle(screen *ebiten.Image, p simulation.Particle) {
	x, y := SimToRenderCoords(p.GetPos())

	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(x-float64(g.particleRadius), y-float64(g.particleRadius))
	screen.DrawImage(g.cachedParticle, &op)
}

func SimToRenderCoords(xSim, ySim float64) (xRender, yRender float64) {
	xRender = xSim * (WindowWidth / simulation.Width)
	yRender = ySim * (WindowHeight / simulation.Height)
	return
}
