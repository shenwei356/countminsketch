package main

import (
	"fmt"
	"os"

	"github.com/shenwei356/countminsketch"
)

func main() {
	var varepsilon, delta float64
	varepsilon, delta = 0.1, 0.9
	s, err := countminsketch.NewWithEstimates(varepsilon, delta)
	checkerr(err)
	fmt.Printf("ε: %f, δ: %f -> d: %d, w: %d\n", varepsilon, delta, s.D(), s.W())

	varepsilon, delta = 0.0001, 0.9999
	s, err = countminsketch.NewWithEstimates(varepsilon, delta)
	checkerr(err)
	fmt.Printf("ε: %f, δ: %f -> d: %d, w: %d\n", varepsilon, delta, s.D(), s.W())

	key := "abc"
	s.UpdateString(key, 1)
	fmt.Printf("%s:%d\n\n", key, s.EstimateString(key))

	//////////////////////////////////////////////////

	file := "data"
	s.UpdateString(key, 2)
	_, err = s.WriteToFile(file)
	defer func() {
		err := os.Remove(file)
		checkerr(err)
	}()

	cm, err := countminsketch.NewFromFile(file)
	checkerr(err)

	fmt.Printf("%s:%d\n", key, cm.EstimateString(key))

	//////////////////////////////////////////////////

	s, err = countminsketch.NewWithEstimates(0.1, 0.9)
	checkerr(err)
	s.UpdateString(key, 10)
	bytes, err := s.MarshalJSON()
	checkerr(err)
	fmt.Println(string(bytes))

	err = s.UnmarshalJSON(bytes)
	checkerr(err)
	s.UpdateString(key, 10)

	fmt.Printf("%s:%d\n", key, s.EstimateString(key))
}

func checkerr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
