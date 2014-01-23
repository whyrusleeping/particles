package main

import (
	"runtime"
	"math/rand"
	"time"
	"flag"
)


var particles = flag.Int("p", 1000, "Number of particles to spawn.")
var srange = flag.Float64("d", 50, "Width of spawn box.")
var svel = flag.Float64("v", 1.5, "Range of initial velocity.")
var smass = flag.Float64("m", 5, "Range of particle mass.")

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(2)
	rand.Seed(time.Now().UnixNano())
	SpawnRange = *srange
	SpawnVel = *svel
	SpawnMass = *smass
	w := NewSim(800,800,*particles)
	w.Run()
}
