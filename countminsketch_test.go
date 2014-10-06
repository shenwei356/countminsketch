package countminsketch

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/bmizerany/assert"
)

// based on https://github.com/jehiah/countmin/blob/master/sketch_test.go
func TestAccuracy(t *testing.T) {
	log.SetOutput(os.Stdout)

	s, err := NewWithEstimates(0.0001, 0.9999)
	if err != nil {
		t.Error(err)
	}
	// s := New(7, 2000)

	iterations := 5500
	var diverged int
	for i := 1; i < iterations; i += 1 {
		v := uint(i % 50)

		s.UpdateString(strconv.Itoa(i), v)
		vv := s.EstimateString(strconv.Itoa(i))
		if vv > v {
			diverged += 1
		}
	}

	var miss int
	for i := 1; i < iterations; i += 1 {
		vv := uint(i % 50)

		v := s.EstimateString(strconv.Itoa(i))
		assert.Equal(t, v >= v, true)
		if v != vv {
			log.Printf("real: %d, estimate: %d\n", vv, v)
			miss += 1
		}
	}
	log.Printf("missed %d of %d (%d diverged during adds)", miss, iterations, diverged)
}

func TestIO(t *testing.T) {
	s, err := NewWithEstimates(0.0001, 0.9999)
	if err != nil {
		t.Error(err)
	}

	iterations := 5500
	cache := make(map[int]uint, iterations)
	for i := 1; i < iterations; i += 1 {
		v := uint(i % 50)
		s.UpdateString(strconv.Itoa(i), v)

		cache[i] = s.EstimateString(strconv.Itoa(i))

	}

	file := "datafile"
	_, err = s.WriteToFile(file)
	defer func() {
		err := os.Remove(file)
		if err != nil {
			t.Error(err)
		}
	}()
	if err != nil {
		t.Error(err)
	}

	cm, err := NewFromFile(file)
	if err != nil {
		t.Error(err)
	}

	for i := 1; i < iterations; i += 1 {
		if cache[i] != cm.EstimateString(strconv.Itoa(i)) {
			t.Error(err)
		}
	}
}

func Benchmark_Update_ε0_001_δ0_999(b *testing.B) {
	s, err := NewWithEstimates(0.001, 0.999)
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < b.N; i++ {
		s.UpdateString(strconv.Itoa(int(rand.Int31())), uint(rand.Int31()))
	}
}

func Benchmark_Estimates_ε0_001_δ0_999(b *testing.B) {
	s, err := NewWithEstimates(0.001, 0.999)
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < b.N; i++ {
		s.EstimateString(strconv.Itoa(int(rand.Int31())))
	}
}

func Benchmark_Update_ε0_000001_δ0_9999(b *testing.B) {
	s, err := NewWithEstimates(0.000001, 0.9999)
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < b.N; i++ {
		s.UpdateString(strconv.Itoa(int(rand.Int31())), uint(rand.Int31()))
	}
}

func Benchmark_Estimates_ε0_000001_δ0_9999(b *testing.B) {
	s, err := NewWithEstimates(0.000001, 0.9999)
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < b.N; i++ {
		s.EstimateString(strconv.Itoa(int(rand.Int31())))
	}
}
