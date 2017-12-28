package cruncher

import (
	"math"
	"math/rand"
	"os"
	"testing"
)

func TestMaxMinMeanMedianAccomulation(t *testing.T) {
	a := NewAccumulator(1000, 5)
	a.Add(1)
	a.Add(2)
	a.Add(4)
	a.Print(os.Stdout)
	intStats := a.GetStats()

	if actual, correct := intStats.Min, int64(1); actual != correct {
		t.Errorf("Min: %d != %d", actual, correct)
	}
	if actual, correct := intStats.Max, int64(4); actual != correct {
		t.Errorf("Max: %d != %d", actual, correct)
	}
	if actual, correct := intStats.Median, int64(2); actual != correct {
		t.Errorf("Median: %d != %d", actual, correct)
	}
	if actual, correct := intStats.Mean, float64(7.0/3.0); actual != correct {
		t.Errorf("Mean: %f != %f", actual, correct)
	}
	if actual, correct := intStats.Count, int64(3); actual != correct {
		t.Errorf("Count: %d != %d", actual, correct)
	}

}

func TestConsecutive(t *testing.T) {
	a := NewAccumulator(1000, 10)
	a.Add(1)
	a.Add(2)
	a.Add(3)
	if len(a.GetStats().GetTermFrequency(10)) != 3 {
		t.Errorf("Should only have 3 terms")
	}
	a.Add(4)
	a.Add(4)
	a.Add(4)
	a.Add(5)
	a.Add(6)
	a.Add(7)
	a.Print(os.Stdout)
	if v := a.GetStats().GetTermFrequency(10)[0].Value; v != 4 {
		t.Errorf("Value should be 4 but was %d.", v)
	}
	testFrequency(t, a.GetStats())
}

func testFrequency(t *testing.T, is IntStats) {
	var prev Pair
	for i, v := range is.GetTermFrequency(10) {
		if i == 0 {
			prev = v
		} else {
			if prev.Frequency < v.Frequency {
				t.Errorf("Term frequency is not in the correct order term %d with value %d and frequency %d should not be after %d with frequency %d",
					i, v.Value, v.Frequency, prev.Value, prev.Frequency)
			}
		}
	}

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
	testFrequency(t, a.GetStats())
	if v := a.GetStats().Median; v != 200 {
		t.Errorf("Median value should be 200 but was %d.", v)
	}

}

func TestSmallAccomulation(t *testing.T) {
	a := NewAccumulator(1000, 5)
	for i := 0; i < 10; i++ {
		a.Add(int64(rand.Int63n(1776) * rand.Int63n(1776)))
	}
	a.Print(os.Stdout)
}

func TestLargeAccomulation(t *testing.T) {
	a := NewAccumulator(1000, 20)
	for i := 0; i < 10000000; i++ {
		a.Add(int64(rand.Int63n(1776) * rand.Int63n(1776)))
	}
	a.Print(os.Stdout)
	testFrequency(t, a.GetStats())
}

func TestGausianAccomulation(t *testing.T) {
	a := NewAccumulator(1000, 5)
	for i := 0; i < 10000000; i++ {
		a.Add(gausian(100, 50))
	}
	// Should have a gausean
	a.Print(os.Stdout)
	testFrequency(t, a.GetStats())
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
