countminsketch
========

An implementation of Count-Min Sketch in Golang.

Introduction of Count-Min Sketch, from Wikipedia[1]

>    The Count–min sketch (or CM sketch) is a probabilistic sub-linear space
>    streaming algorithm which can be used to summarize a data stream in many
>    different ways. The algorithm was invented in 2003 by Graham Cormode and
>    S. Muthu Muthukrishnan.
>    
>    Count–min sketches are somewhat similar to Bloom filters; the main
>    distinction is that Bloom filters represent sets, while CM sketches
>    represent multisets and frequency tables. Spectral Bloom filters with
>    multi-set policy, are conceptually isomorphic to the Count-Min Sketch.

The code is deeply inspired by an implementation of Bloom filters in golang,
[bloom](https://github.com/willf/bloom).

Same to bloom, the hashing function used is FNV, provided by Go package
(hash/fnv). For a item, the 64-bit FNV hash is computed, and upper and lower
32 bit numbers, call them _h1_ and _h2_, are used. Then, the _i_ th hashing
function is:

    h1 + h2*i

Sketch Accuracy
-------------

Accuracy guarantees will be made in terms of a pair of user specified parameters,
ε and δ, meaning that the error in answering a query is within a factor of ε with
 probability δ[2]

For a sketch of size _w_ × _d_ with total count _N_ , it follows that any
estimate has error at most _2N/w_, with probability at least 1 - (1/2)^_d_.
So setting the parameters _w_ and _d_ large enough allows us to achieve
very high accuracy while using relatively little space[3].

Suppose we want an error of at most 0.1% (of the sum of all frequencies),
with 99.9% certainty. Then we want 2/_w_ = 1/1000, we set _w_ = 2000,
and = 0.001, i.e. _d_ = log 0.001 / log 0.5 ≤ 10. Using uint counters,
the space required by the array of counters is _w_ × _d_ × 4 = 80KB in 32 bit
OS, and _w_ × _d_ × 8 = 160KB in 64 bit OS [3].

To create with given error rate and confidence, we could use constructor NewWithEstimates.

Parallelization
-----------

The parallelizing part of Count-Min Sketch is the hashing step. But in this implementation,
only one basic hashing step is computed. So the parallelization is not necessary.

If you have to, try to split the data and count separately. And at last `Merge` them.

Install
-------

This package is "go-gettable", just:

    go get github.com/shenwei356/countminsketch

Usage
-------------

    import "github.com/shenwei356/countminsketch"

	var epsilon, delta float64
	epsilon, delta = 0.1, 0.9
	s := countminsketch.NewWithEstimates(epsilon, delta)
	fmt.Printf("ε: %f, δ: %f -> d: %d, w: %d\n", epsilon, delta, s.D(), s.W())

	epsilon, delta = 0.0001, 0.9999
	s = countminsketch.NewWithEstimates(epsilon, delta)
	fmt.Printf("ε: %f, δ: %f -> d: %d, w: %d\n", epsilon, delta, s.D(), s.W())

	key := "abc"
	s.UpdateString(key, 1)
	fmt.Printf("%s:%d\n\n", key, s.EstimateString(key))

	//////////////////////////////////////////////////
	file := "data"
	s.UpdateString(key, 2)
	_, err := s.WriteToFile(file)
	defer func() {
		err := os.Remove(file)
		checkerr(err)
	}()

	cm, err := countminsketch.NewFromFile(file)
	checkerr(err)

	fmt.Printf("%s:%d\n", key, cm.EstimateString(key))

    //////////////////////////////////////////////////
	s = countminsketch.NewWithEstimates(0.1, 0.9)
	s.UpdateString(key, 10)
	bytes, err := s.MarshalJSON()
	checkerr(err)
	fmt.Println(string(bytes))

	err = s.UnmarshalJSON(bytes)
	checkerr(err)
	s.UpdateString(key, 10)

	fmt.Printf("%s:%d\n", key, s.EstimateString(key))

Output

    ε: 0.100000, δ: 0.900000 -> d: 4, w: 20
    ε: 0.000100, δ: 0.999900 -> d: 14, w: 20000
    abc:1

    abc:3
    {"d":4,"w":20,"count":[[0,0,0,10,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,10,0,0,0,0],[0,0,0,0,0,0,0,10,0,0,0,0,0,0,0,0,0,0,0,0],[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,10]]}
    abc:20

Benchmark
--------

    Benchmark_Update_ε0_001_δ0_999           5000000               515 ns/op
    Benchmark_Estimates_ε0_001_δ0_999        5000000               481 ns/op
    Benchmark_Update_ε0_000001_δ0_9999       2000000               941 ns/op
    Benchmark_Estimates_ε0_000001_δ0_9999    2000000               841 ns/op


Documentation
-------------

[![GoDoc](https://godoc.org/github.com/shenwei356/countminsketch?status.svg)](https://godoc.org/github.com/shenwei356/countminsketch)

Reference
-------------
1. [Wikipedia](http://en.wikipedia.org/wiki/Count%E2%80%93min_sketch)
2. [An Improved Data Stream Summary: The Count-Min Sketch and its Applications](http://www.cse.unsw.edu.au/~cs9314/07s1/lectures/Lin_CS9314_References/cm-latin.pdf)
3. [Approximating Data with the Count-Min Data Structure](http://dimacs.rutgers.edu/~graham/pubs/papers/cmsoft.pdf)
4. [https://github.com/jehiah/countmin](https://github.com/jehiah/countmin)
5. [https://github.com/mtchavez/countmin](https://github.com/mtchavez/countmin)

Copyright
--------

Copyright (c) 2014-2016, Wei Shen (shenwei356@gmail.com)

[MIT License](https://github.com/shenwei356/countminsketch/blob/master/LICENSE)
