package main

import (
	"runtime"
	"math/rand"
	"time"
	"flag"
)


var threads = flag.Int("t", 2, "Number of threads to use.")
var particles = flag.Int("p", 1000, "Number of particles to spawn.")

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(6)
	rand.Seed(time.Now().UnixNano())
	w := NewSim(800,800,*particles,*threads)
	w.Run()
}
