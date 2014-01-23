package main

import (
	"fmt"
	"github.com/whyrusleeping/sdl"
	"runtime"
	"math"
	"math/rand"
	"time"
	"image/color"
)

var SpawnRange float64 = 50
var SpawnVel float64 = 1.5
var SpawnMass float64 = 5

type Coord3 struct {
	X,Y,Z float64
}

func (c Coord3) Add(o Coord3) Coord3 {
	return Coord3{c.X + o.X,
	c.Y + o.Y,
	c.Z + o.Z}
}

func (c Coord3) VecLen() float64 {
	return math.Sqrt(c.X*c.X+c.Y*c.Y+c.Z*c.Z)
}

func (c Coord3) Sub(o Coord3) Coord3 {
	return Coord3{c.X - o.X,
	c.Y - o.Y,
	c.Z - o.Z}
}

func (c *Coord3) AddInPlace(o Coord3) {
	c.X += o.X
	c.Y += o.Y
	c.Z += o.Z
}

func (c Coord3) Mul(v float64) Coord3 {
	return Coord3{c.X * v,
	c.Y * v,
	c.Z * v}
}

func (c Coord3) Div(v float64) Coord3 {
	return Coord3{c.X / v,
	c.Y / v,
	c.Z / v}
}

func (c Coord3) Dist(o Coord3) float64 {
	return math.Sqrt(math.Pow(c.X - o.X,2) +
	math.Pow(c.Y - o.Y,2) +
	math.Pow(c.Z - o.Z,2))
}


type Particle struct {
	vX, vY, vZ float64
	pX, pY, pZ float64
	Mass float64
}

type Simulation struct {
	particles []Particle
	running bool
	X,Y int
	screen *sdl.Display
	bg color.RGBA
	events chan sdl.Event
	screenRect sdl.Rect
	nThreads int

	clp *CLSetup

	oX,oY int
	scale float64
	deltaT float64
	bigG float64
}

func RandRange(rng float64) float64 {
	return ((2 * rng) * (float64(rand.Uint32()) / float64(math.MaxUint32))) - rng
}

func RandCoord3(rng float64) Coord3 {
	return Coord3{RandRange(rng),RandRange(rng),RandRange(rng)}
}

func RandParticle() Particle {
	p := Particle{}
	Loc := RandCoord3(SpawnRange)
	p.pX = Loc.X
	p.pY = Loc.Y
	p.pZ = Loc.Z
	p.Mass = RandRange(SpawnMass) + SpawnMass
	Vel := RandCoord3(SpawnVel)
	p.vX = Vel.X
	p.vY = Vel.Y
	p.vZ = Vel.Z
	return p
}

func NewSim(x,y int, Particles int) *Simulation {
	w := new(Simulation)
	w.X = x
	w.Y = y
	w.oX = x / 2
	w.oY = y / 2
	w.scale = 1
	w.screenRect.X = sdl.Int(x)
	w.screenRect.Y = sdl.Int(y)
	w.events = make(chan sdl.Event)
	for i := 0; i < Particles; i++ {
		w.particles = append(w.particles, RandParticle())
	}

	w.deltaT = 1
	w.bigG = 3.0

	return w
}

//This function is organized so that the most intensive calculations 
//will occur during the previous render cycle, thereby maximizing
//the amount of wasted time spent waiting
func (s *Simulation) UpdateParticles() {
	//Velocities can be updated asynchronously
	s.clp.Execute(s.particles)
	for n := 0; n < len(s.particles); n++ {
		p := s.particles[n]
		s.particles[n].pX += s.deltaT * p.vX
		s.particles[n].pY += s.deltaT * p.vY
		s.particles[n].pZ += s.deltaT * p.vZ
	}
}

//Render the grid on the screen
func (w *Simulation) DrawParticles() {
	//Fill the background with a set color
	w.screen.SetDrawColor(w.bg)
	w.screen.DrawRect(w.screenRect)
	w.screen.Clear()
	w.screen.SetDrawColor(color.RGBA{255,255,255,0})
	for i := 0; i < len(w.particles); i++ {
		w.screen.DrawPoint(int(w.particles[i].pX / w.scale) + w.oX,int(w.particles[i].pY / w.scale) + w.oY)
	}
	//Put all this on the screen now
	w.screen.Present()
}

func FpsTicker(tick chan bool) {
	frames := float64(0)
	gran := 5
	timer := time.Tick(time.Second * time.Duration(gran))
	for {
		select {
		case <-tick:
			frames++
		case <-timer:
			fmt.Printf("%f fps.\n", frames/float64(gran))
			frames = 0
		}
	}
}

func (s *Simulation) HandleKey(ev *sdl.KeyboardEvent) {
	//fmt.Printf("%d %d\n", ev.State, ev.Keysym.Sym)
	if ev.State == 1 {
		switch ev.Keysym.Sym {
		case sdl.K_w:
			fmt.Println("Move up.")
			s.oY += 10
		case sdl.K_a:
			fmt.Println("Move left.")
			s.oX += 10
		case sdl.K_s:
			fmt.Println("Move down.")
			s.oY -= 10
		case sdl.K_d:
			fmt.Println("Move right.")
			s.oX -= 10
		case sdl.K_j:
			fmt.Println("Zoom in!")
			s.scale /= 1.1
		case sdl.K_k:
			fmt.Println("Zoom out!")
			s.scale *= 1.1
		}
	}
}

func (w *Simulation) Run() {
	//Lock the thread because SDL uses openGL and openGL cant be
	//called across threads
	runtime.LockOSThread()
	err := sdl.Init(sdl.INIT_EVERYTHING)

	if err != nil {
		//If this fails, we kinda want to die for now
		panic(err)
	}

	//If this function quits unexpectedly, shut down SDL
	defer sdl.Quit()

	//Make a new screen object to render to
	w.screen, err = sdl.NewDisplay(w.X, w.Y, sdl.WINDOW_OPENGL)
	if err != nil {
		panic(err)
	}

	w.screen.SetTitle("Awesome Simulation Title Here")

	w.clp = StartupCL(w.particles)

	w.running = true
	tick := make(chan bool)
	go FpsTicker(tick)
	for {
		for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
			switch ev := ev.(type) {
			case *sdl.QuitEvent:
				w.running = false
				return
			case *sdl.KeyboardEvent:
				w.HandleKey(ev)
			}
		}
		w.UpdateParticles()
		w.DrawParticles()
		tick <- true
	}
}
