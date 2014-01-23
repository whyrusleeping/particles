package main

import (
	"fmt"
	"github.com/whyrusleeping/sdl"
	"runtime"
	"math"
	"math/rand"
	"time"
	"image/color"
	"sync"
)

var SpawnRange float64 = 50
var SpawnVel float64 = 1.5
var SpawnMass float64 = 5

type Coord3 struct {
	X,Y,Z float64
}

func (c Coord3) Add(o Coord3) Coord3 {
	return Coord3{c.X + o.X, c.Y + o.Y, c.Z + o.Z}
}

func (c Coord3) VecLen() float64 {
	return math.Sqrt(c.X*c.X+c.Y*c.Y+c.Z*c.Z)
}

func (c *Coord3) Wrap(dist float64) {
	if c.X > dist {
		c.X -= 2*dist
	}
	if c.X * -1 > dist {
		c.X += 2*dist
	}
	if c.Y > dist {
		c.Y -= 2*dist
	}
	if c.Y * -1 > dist {
		c.Y += 2*dist
	}
	if c.Z > dist {
		c.Z -= 2*dist
	}
	if c.Z * -1 > dist {
		c.Z += 2*dist
	}
}

func (c *Coord3) Cap(max float64) {
	v := c.VecLen()
	if v > max {
		r := max / v
		c.X *= r
		c.Y *= r
		c.Z *= r
	}
}

func (c Coord3) Sub(o Coord3) Coord3 {
	return Coord3{c.X - o.X, c.Y - o.Y, c.Z - o.Z}
}

func (c *Coord3) AddInPlace(o Coord3) {
	c.X += o.X
	c.Y += o.Y
	c.Z += o.Z
}

func (c Coord3) Mul(v float64) Coord3 {
	return Coord3{c.X * v, c.Y * v, c.Z * v}
}

func (c Coord3) Div(v float64) Coord3 {
	return Coord3{c.X / v, c.Y / v, c.Z / v}
}

func (c Coord3) Dist(o Coord3) float64 {
	fmt.Println("Use sub and VecLen instead.")
	return math.Sqrt(math.Pow(c.X - o.X,2) +
					 math.Pow(c.Y - o.Y,2) +
					 math.Pow(c.Z - o.Z,2))
}


type Particle struct {
	Mass float64
	Loc Coord3
	Vel Coord3
	Color color.RGBA
}

func (p *Particle) Collide(o *Particle) {
	mom := p.Vel.Mul(p.Mass)
	mom.AddInPlace(o.Vel.Mul(o.Mass))
	mom = mom.Div(2)
	p.Vel = mom.Div(p.Mass)
	o.Vel = mom.Div(o.Mass)
/* More correct, doesnt work
	mom := p.Vel.Mul(p.Mass)
	mom.AddInPlace(o.Vel.Mul(o.Mass))
	msum := p.Mass + o.Mass
	mom = mom.Div(msum)
	p.Vel = mom.Mul(p.Mass)
	o.Vel = mom.Mul(o.Mass)
	*/
}

type Simulation struct {
	particles []*Particle
	running bool
	X,Y int
	screen *sdl.Display
	bg color.RGBA
	events chan sdl.Event
	screenRect sdl.Rect
	nThreads int

	start chan struct{}
	wgAcc sync.WaitGroup
	wgPos sync.WaitGroup

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

func RandParticle() *Particle {
	p := new(Particle)
	p.Loc = RandCoord3(SpawnRange)
	p.Mass = RandRange(SpawnMass) + SpawnMass
	p.Vel = RandCoord3(SpawnVel)
	return p
}

func NewSim(x,y int, Particles int, Threads int) *Simulation {
	w := new(Simulation)
	w.X = x
	w.Y = y
	w.oX = x / 2
	w.oY = y / 2
	w.scale = 1
	w.screenRect.X = sdl.Int(x)
	w.screenRect.Y = sdl.Int(y)
	w.events = make(chan sdl.Event)
	for i := 0; i < 1; i++ {
		heavy := RandParticle()
		heavy.Mass = 500000
		heavy.Vel = Coord3{}
		w.particles = append(w.particles, heavy)
	}
	for i := 0; i < Particles; i++ {
		w.particles = append(w.particles, RandParticle())
	}
	w.nThreads = Threads
	w.start = make(chan struct{})

	w.deltaT = 0.1
	w.bigG = 3.0

	for i := 0; i < Threads; i++ {
		go w.UpdateRoutineAdv(i * (len(w.particles) / w.nThreads), (i+1) * (len(w.particles) / w.nThreads))
	}
	return w
}

//A single thread for updating particle velocity/location in parallel
func (s *Simulation) UpdateRoutine(beg, end int) {
	fmt.Printf("Range: %d to %d\n", beg, end)
	for {
		<-s.start
		s.wgAcc.Add(1)
		for n,cur := range s.particles[beg:end] {
			for j,p := range s.particles {
				if n == j {
					continue
				}
				dvec := p.Loc.Sub(cur.Loc)
				dist := dvec.VecLen()
				if dist < 0.000001 {
					dist = 0.000001
				}
				acc := s.bigG * p.Mass / (dist * dist)
				aVec := dvec.Mul(s.deltaT * acc/dist)
				cur.Vel.AddInPlace(aVec)
			}
		}
		s.wgAcc.Done()
		s.wgAcc.Wait()

		//Once all velocities have been updated, update location
		s.wgPos.Add(1)
		for _,p := range s.particles[beg:end] {
			p.Loc.AddInPlace(p.Vel.Mul(s.deltaT))
		}
		s.wgPos.Done()
	}
}

//A more advanced particle updater that takes collisions into account
func (s *Simulation) UpdateRoutineAdv(beg, end int) {
	fmt.Printf("Range: %d to %d\n", beg, end)
	for {
		<-s.start
		s.wgAcc.Add(1)
		for n,cur := range s.particles[beg:end] {
			for j,p := range s.particles {
				if n == j {
					continue
				}
				dvec := p.Loc.Sub(cur.Loc)
				dist := dvec.VecLen()
				if dist < 1 {
					//Collision
					cur.Collide(p)
					continue
				}
				acc := s.bigG * p.Mass / (dist * dist)
				aVec := dvec.Mul(s.deltaT * acc/dist)
				cur.Vel.AddInPlace(aVec)
			}
		}
		s.wgAcc.Done()
		s.wgAcc.Wait()

		//Once all velocities have been updated, update location
		s.wgPos.Add(1)
		for _,p := range s.particles[beg:end] {
			p.Vel.Cap(300)
			p.Loc.AddInPlace(p.Vel.Mul(s.deltaT))
			//p.Loc.Wrap(1000)
		}
		s.wgPos.Done()
	}
}

//This function is organized so that the most intensive calculations 
//will occur during the previous render cycle, thereby maximizing
//the amount of wasted time spent waiting
func (w *Simulation) UpdateParticles() {
	//Velocities can be updated asynchronously
	w.wgAcc.Wait()
	w.wgPos.Wait()
	for i := 0; i < w.nThreads; i++ {
		w.start <- struct{}{}
	}
}

//Render the grid on the screen
func (w *Simulation) DrawParticles() {
	//Fill the background with a set color
	w.screen.SetDrawColor(w.bg)
	w.screen.DrawRect(w.screenRect)
	w.screen.Clear()
	for i := 0; i < len(w.particles); i++ {
		w.screen.DrawPoint(int(w.particles[i].Loc.X / w.scale) + w.oX,int(w.particles[i].Loc.Y / w.scale) + w.oY)
		w.screen.SetDrawColor(w.particles[i].Color)
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


	for i := 0; i < w.nThreads; i++ {
		w.start <-struct{}{}
	}

	w.running = true
	tick := make(chan bool)
	go FpsTicker(tick)
	drag := false
	for {
		for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
			switch ev := ev.(type) {
			case *sdl.QuitEvent:
				w.running = false
				return
			case *sdl.KeyboardEvent:
				w.HandleKey(ev)
			case *sdl.MouseButtonEvent:
				if ev.Button == 1 {
					drag = !drag
					fmt.Println(drag)
				}
				fmt.Println(ev.Button)
			case *sdl.MouseMotionEvent:
				if drag {
					w.oX += ev.Xrel
					w.oY += ev.Yrel
				}
			case *sdl.MouseWheelEvent:
				fmt.Println("Wheeel...")
			default:
				fmt.Println("Uncaught event!")
			}
		}
		w.UpdateParticles()
		w.DrawParticles()
		tick <- true
	}
}
