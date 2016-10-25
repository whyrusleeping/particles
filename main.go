package main

import (
	"flag"
	"math/rand"
	"time"
)

var threads = flag.Int("t", 1, "Number of threads to use.")
var particles = flag.Int("p", 1000, "Number of particles to spawn.")
var srange = flag.Float64("d", 50, "Width of spawn box.")
var svel = flag.Float64("v", 1.5, "Range of initial velocity.")
var smass = flag.Float64("m", 5, "Range of particle mass.")

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	SpawnRange = *srange
	SpawnVel = *svel
	SpawnMass = *smass
	w := NewSim(1600, 1600, *particles, *threads)
	w.Run()
}
