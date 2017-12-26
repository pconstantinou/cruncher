/*
Package cruncher provides a quick way to acquire detailed statistics on
a dataset of arbitrary size.
Usage:
    a := NewAccumulator(1000,10)
    while (dataAvailable) {
      a.Add(integer)
    }
    a.Summarize() // This must be called finalize the computation of the median/mean
    a.Print(os.StdOut)
    fmt.Printf("Median: %d", a.IntStats.Media)
Median value is approximated using the approach defined here:
http://web.ipac.caltech.edu/staff/fmasci/home/astro_refs/Remedian.pdf
While this package will work well for data sets of any size it's designed to scale to large
quantities of data this the reliance on int64.

Below is an example of the output of the data gathered from a random gausian number
generator.

	= Summary ======================
	Min              -150
	Max               354
	Count        10000000
	Mean              100.000
	Median            100

	= Distribution (interval: 63) ====
	    -150 -      -56 :    8947 (0.09%)**
	     -55 -        7 :  305232 (3.05%)
	       8 -       70 : 2428044 (24.28%)
	      71 -      133 : 4774283 (47.74%)
	     134 -      196 : 2221206 (22.21%)
	     197 -      259 :  255458 (2.55%)
	     261 -      354 :    6829 (0.07%)**

	= Top Value Frequency ==========
	 1.      100 :  159908 (1.60%)
	 2.      102 :   79795 (0.80%)
	 3.       98 :   79726 (0.80%)
	 4.      106 :   79595 (0.80%)
	 5.      105 :   79553 (0.80%)

*/
package cruncher

import (
	"fmt"
	"io"
	"math"
	"sort"
)

const (
	// InitialRemedianSize is the number of entries pre-allocated for maintaining
	// the median/
	InitialRemedianSize = 4
)

// IntStats contains all the stats accumulated. It's best to
// maintain references only to the IntStats once the accumulation is
// complete and remove references to Accumulator.
type IntStats struct {
	// Smallest valued added
	Min int64
	// Largest value added
	Max int64
	// Number of entries added
	Count int64
	// Mean is computed using a total / count it may be subject to overflow
	Mean float64
	// Median is an approximation using the Remedian technicque
	Median int64
	// FrequencyDistribution contains the count of occurances within a bucket
	FrequencyDistribution []int64
	// BucketSize contains the range of values within a bucket
	BucketSize int64
	// FrequencyDistributionStartingValue is the starting value for the
	// frequency distribution. Distributions don't have to start at zero
	FrequencyDistributionStartingValue int64
	// OutlierBefore is the number of occurances lower than FrequencyDistributionStartingValue
	OutlierBefore int64
	// OutlierAfter is the number of occurances higher than the largest bucket
	OutlierAfter int64
	// Frequency
	ValueFrequency map[int64]int64
}

// Accumulator maintains the transient state collected when accomulating
// statistics on a set of data. The results are available GetStats
type Accumulator struct {
	intStats           IntStats
	remedians          [][]int64
	total              int64
	appoximationWindow int
	buckets            int
}

// NewAccumulator allocates an accumulator that collects statistics on data added.
// appoximationWindow is the amount of data to sample before computing
// the min and max for the frequency distribution. This
// value is also used to compute the median. Larger values require more
// memory but may be required if data values are not
// randomly distributed.
// buckets are the number of groups in the frequency distribution
func NewAccumulator(appoximationWindow, buckets int) *Accumulator {
	a := new(Accumulator)
	a.appoximationWindow = appoximationWindow
	a.remedians = make([][]int64, 0, InitialRemedianSize)
	a.buckets = buckets
	return a
}

// Add adds a value to the data set to be summarized. Add is typically a constant
// time operation but may periodically include some iteration to update some
// statistics.
func (a *Accumulator) Add(value int64) {
	// Adjust Min and Max
	if a.intStats.Count == 0 {
		a.intStats.Max = value
		a.intStats.Min = value
		a.intStats.ValueFrequency = make(map[int64]int64)
	} else {
		if a.intStats.Max < value {
			a.intStats.Max = value
		} else if a.intStats.Min > value {
			a.intStats.Min = value
		}
	}
	// Adjust Counts and Totals
	a.intStats.Count++
	a.total += value

	// Update frequency distribution
	count := a.intStats.Count

	// One time configure Frequency Distribution
	if len(a.intStats.FrequencyDistribution) > 0 {
		a.incrementFrequencyDistribution(value)
	} else if count == int64(a.appoximationWindow) {
		a.initializeFrequencyDistribution()
	}
	// Must do this last so the full set of values is available
	a.pushMedianValue(0, value)

	// Count frequencies but don't counnt more than a.appoximationWindow
	valueCount, present := a.intStats.ValueFrequency[value]
	if present {
		a.intStats.ValueFrequency[value] = valueCount + 1
	} else if len(a.intStats.ValueFrequency) < a.appoximationWindow {
		a.intStats.ValueFrequency[value] = 1
	}
}

func (a *Accumulator) initializeFrequencyDistribution() {
	a.intStats.OutlierAfter = 0
	a.intStats.OutlierBefore = 0
	a.intStats.FrequencyDistribution = make([]int64, a.buckets)
	a.intStats.FrequencyDistributionStartingValue = a.intStats.Min
	diff := a.intStats.Max - a.intStats.Min
	a.intStats.BucketSize = int64(math.Ceil(float64(diff+1) / float64(a.buckets)))
	for _, v := range a.remedians[0] {
		a.incrementFrequencyDistribution(int64(v))
	}
}

func (a *Accumulator) incrementFrequencyDistribution(value int64) (offset int) {
	// Update bucket value
	offset = int(math.Floor((float64(value-a.intStats.FrequencyDistributionStartingValue) / float64(a.intStats.BucketSize))))
	// Handle out of bounds
	if offset < 0 {
		a.intStats.OutlierBefore++
	} else if offset >= len(a.intStats.FrequencyDistribution) {
		a.intStats.OutlierAfter++
	} else {
		// Increment bucket
		a.intStats.FrequencyDistribution[offset]++
	}
	return offset
}

type int64arr []int64

func (a int64arr) Len() int           { return len(a) }
func (a int64arr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a int64arr) Less(i, j int) bool { return a[i] < a[j] }

func (a *Accumulator) pushMedianValue(offset int, value int64) (computed bool, min, max, median int64) {
	if len(a.remedians) <= offset {
		a.remedians = append(a.remedians, make([]int64, 0, a.appoximationWindow))
	}
	a.remedians[offset] = append(a.remedians[offset], value)
	if medianLength := len(a.remedians[offset]); a.appoximationWindow < medianLength {
		min, max, median = computeMedian(a.remedians[offset])
		computed = true
		a.pushMedianValue(offset+1, median)
		a.remedians[offset] = a.remedians[offset][:0]
	}
	return computed, min, max, median
}

func computeMedian(values []int64) (min, max, median int64) {
	sort.Sort(int64arr(values))
	l := len(values)
	return values[0], values[l-1], values[l/2]
}

// Summarize computes the frequency distribution and median
// calculation on the data samples that haven't been summarized
// yet.
func (a *Accumulator) Summarize() {
	if a.intStats.Count < int64(a.appoximationWindow) {
		a.initializeFrequencyDistribution()
	}
	a.intStats.Mean = (float64)(a.total / a.intStats.Count)
	for i := len(a.remedians) - 1; i >= 0; i-- {
		_, _, a.intStats.Median = computeMedian(a.remedians[i])
		return
	}
}

// GetTermFrequency returns the most frequently used terms. This is an
// Approximation. If the first term does not appear within the
// first approximationWindow data set then it will be omitted from the results
func (is IntStats) GetTermFrequency(topN int) PairList {
	pl := make(PairList, len(is.ValueFrequency))
	if topN > len(is.ValueFrequency) {
		topN = len(is.ValueFrequency)
	}
	i := 0
	for k, f := range is.ValueFrequency {
		pl[i] = Pair{k, f}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl[:topN]
}

// Pair provides a touple of the value provide and the frequency of the values use
type Pair struct {
	Value     int64
	Frequency int64
}

// PairList is an array of Pair's
type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Frequency < p[j].Frequency }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// GetStats provides the current stats accumulated. If the data set continues to
// accumulate the accumulator update the results however,
// The copy returned will not be impacted.
func (a *Accumulator) GetStats() IntStats {
	a.Summarize()
	return a.intStats
}

// Print an ascii formatted human readable version of the summarized data
func (a *Accumulator) Print(w io.Writer) {
	a.Summarize()
	a.intStats.Print(w)
}

// Print outputs all the the acquired data about the accumulated values.
func (is IntStats) Print(w io.Writer) {
	fmt.Fprintf(w, "= Summary ======================\n")
	fmt.Fprintf(w, "%-8s %12d\n", "Min", is.Min)
	fmt.Fprintf(w, "%-8s %12d\n", "Max", is.Max)
	fmt.Fprintf(w, "%-8s %12d\n", "Count", is.Count)
	fmt.Fprintf(w, "%-8s %16.3f\n", "Mean", is.Mean)
	fmt.Fprintf(w, "%-8s %12d\n", "Median", is.Median)

	fmt.Println()
	fmt.Fprintf(w, "= Distribution (interval: %d) ====\n", is.BucketSize)
	if is.OutlierBefore > 0 {
		fmt.Fprintf(w, "%8d - %8d :%8d (%4.2f%%)**\n", is.Min, is.FrequencyDistributionStartingValue-1,
			is.OutlierBefore, 100.0*float64(is.OutlierBefore)/float64(is.Count))
	}

	for key, value := range is.FrequencyDistribution {
		fmt.Fprintf(w, "%8d - %8d :%8d (%4.2f%%)\n",
			(is.FrequencyDistributionStartingValue)+(is.BucketSize*int64(key)),
			((is.FrequencyDistributionStartingValue)+(is.BucketSize*(int64(key)+1)))-1, value,
			100.0*float64(value)/float64(is.Count))
	}
	if is.OutlierAfter > 0 {
		fmt.Fprintf(w, "%8d - %8d :%8d (%4.2f%%)**\n",
			is.FrequencyDistributionStartingValue+(is.BucketSize*int64(len(is.FrequencyDistribution)))+1,
			is.Max, is.OutlierAfter, 100.0*float64(is.OutlierAfter)/float64(is.Count))
	}
	fmt.Println()
	fmt.Fprintf(w, "= Top Value Frequency ==========\n")
	for i, pair := range is.GetTermFrequency(5) {
		fmt.Fprintf(w, "%2d. %8d :%8d (%4.2f%%)\n", i+1, pair.Value, pair.Frequency,
			100.0*float64(pair.Frequency)/float64(is.Count))
	}
	fmt.Println()
}
