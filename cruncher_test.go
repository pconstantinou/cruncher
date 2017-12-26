package cruncher

import (
	"math"
	"math/rand"
	"os"
	"testing"
)

func TestSmallAccomulation(t *testing.T) {
	a := NewAccumulator(1000, 5)
	for i := 0; i < 10; i++ {
		a.Add(int64(rand.Int63n(1776) * rand.Int63n(1776)))
	}
	a.Print(os.Stdout)
}

func TestFixed(t *testing.T) {
	a := NewAccumulator(1000, 10)
	a.Add(200)
	a.Add(1000)
	a.Print(os.Stdout)
	a.Add(100)
	a.Add(200)
	a.Add(200)
	a.Add(1000)
	a.Add(1000)
	a.Print(os.Stdout)
}

func TestLargeAccomulation(t *testing.T) {
	a := NewAccumulator(1000, 20)
	for i := 0; i < 100000000; i++ {
		a.Add(int64(rand.Int63n(1776) * rand.Int63n(1776)))
	}
	// Should have an even distribution
	a.Print(os.Stdout)
}

func TestGausianAccomulation(t *testing.T) {
	a := NewAccumulator(1000, 5)
	for i := 0; i < 10000000; i++ {
		a.Add(gausian(100, 50))
	}
	// Should have a gausean
	a.Print(os.Stdout)
}

func BenchmarkGausianAccomulation(b *testing.B) {
	a := NewAccumulator(1000, 10)
	for i := 0; i < 100000*b.N; i++ {
		a.Add(gausian(100, 50))
	}
	a.Print(os.Stdout)
}

var y2 float64
var useLast = false

func gausian(mean int, standardDeviation float64) int64 {
	var x1, x2, w, y1 float64

	if useLast {
		y1 = y2
		useLast = false
	} else {
		w = 2
		for w >= 1.0 {
			x1 = 2.0*rand.Float64() - 1.0
			x2 = 2.0*rand.Float64() - 1.0
			w = x1*x1 + x2*x2
		}
		w = math.Sqrt((-2.0 * math.Log(w)) / w)
		y1 = x1 * w
		y2 = x2 * w
		useLast = true
	}

	return int64(mean) + int64(y1*standardDeviation)

}
