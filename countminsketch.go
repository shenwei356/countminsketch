/*Package countminsketch is an implementation of Count-Min Sketch in Golang.

http://github.com/shenwei356/countmin/

The code is deeply inspired by an implementation of Bloom filters in golang,
[bloom](https://github.com/willf/bloom).
*/
package countminsketch

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash"
	"hash/fnv"
	"io"
	"math"
	"os"
)

// CountMinSketch struct. d is the number of hashing functions,
// w is the size of every hash table.
// count, a matrix, is used to store the count.
// uint is used to store count, the maximum count is 1<<32-1 in
// 32 bit OS, and 1<<64-1 in 64 bit OS.
type CountMinSketch struct {
	d      uint
	w      uint
	count  [][]uint64
	hasher hash.Hash64
}

// New creates a new Count-Min Sketch with _d_ hashing functions
// and _w_ hash value range
func New(d uint, w uint) (s *CountMinSketch, err error) {
	if d <= 0 || w <= 0 {
		return nil, errors.New("countminsketch: values of d and w should both be greater than 0")
	}

	s = &CountMinSketch{
		d:      d,
		w:      w,
		hasher: fnv.New64(),
	}
	s.count = make([][]uint64, d)
	for r := uint(0); r < d; r++ {
		s.count[r] = make([]uint64, w)
	}

	return s, nil
}

// NewWithEstimates creates a new Count-Min Sketch with given error rate and confidence.
// Accuracy guarantees will be made in terms of a pair of user specified parameters,
// ε and δ, meaning that the error in answering a query is within a factor of ε with
// probability δ
func NewWithEstimates(epsilon, delta float64) (*CountMinSketch, error) {
	if epsilon <= 0 || epsilon >= 1 {
		return nil, errors.New("countminsketch: value of epsilon should be in range of (0, 1)")
	}
	if delta <= 0 || delta >= 1 {
		return nil, errors.New("countminsketch: value of delta should be in range of (0, 1)")
	}

	w := uint(math.Ceil(2 / epsilon))
	d := uint(math.Ceil(math.Log(1-delta) / math.Log(0.5)))
	// fmt.Printf("ε: %f, δ: %f -> d: %d, w: %d\n", epsilon, delta, d, w)
	return New(d, w)
}

// NewFromFile creates a new Count-Min Sketch from dumpped file
func NewFromFile(file string) (*CountMinSketch, error) {
	s, err := New(1, 1)
	if err != nil {
		return nil, err
	}
	_, err = s.ReadFromFile(file)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// D returns the number of hashing functions
func (s *CountMinSketch) D() uint {
	return s.d
}

// W returns the size of hashing functions
func (s *CountMinSketch) W() uint {
	return s.w
}

// get the two basic hash function values for data.
// Based on https://github.com/willf/bloom/blob/master/bloom.go
func (s *CountMinSketch) baseHashes(key []byte) (a uint32, b uint32) {
	s.hasher.Reset()
	s.hasher.Write(key)
	sum := s.hasher.Sum(nil)
	upper := sum[0:4]
	lower := sum[4:8]
	a = binary.BigEndian.Uint32(lower)
	b = binary.BigEndian.Uint32(upper)
	return
}

// Get the _w_ locations to update/Estimate
// Based on https://github.com/willf/bloom/blob/master/bloom.go
func (s *CountMinSketch) locations(key []byte) (locs []uint) {
	locs = make([]uint, s.d)
	a, b := s.baseHashes(key)
	ua := uint(a)
	ub := uint(b)
	for r := uint(0); r < s.d; r++ {
		locs[r] = (ua + ub*r) % s.w
	}
	return
}

// Update the frequency of a key
func (s *CountMinSketch) Update(key []byte, count uint64) {
	for r, c := range s.locations(key) {
		s.count[r][c] += count
	}
}

// UpdateString updates the frequency of a key
func (s *CountMinSketch) UpdateString(key string, count uint64) {
	s.Update([]byte(key), count)
}

// Estimate the frequency of a key. It is point query.
func (s *CountMinSketch) Estimate(key []byte) uint64 {
	var min uint64
	for r, c := range s.locations(key) {
		if r == 0 || s.count[r][c] < min {
			min = s.count[r][c]
		}
	}
	return min
}

// EstimateString estimate the frequency of a key of string
func (s *CountMinSketch) EstimateString(key string) uint64 {
	return s.Estimate([]byte(key))
}

// Merge combines this CountMinSketch with another one
func (s *CountMinSketch) Merge(other *CountMinSketch) error {
	if s.d != other.d {
		return errors.New("countminsketch: matrix depth must match")
	}

	if s.w != other.w {
		return errors.New("countminsketch: matrix width must match")
	}

	for i := uint(0); i < s.d; i++ {
		for j := uint(0); j < s.w; j++ {
			s.count[i][j] += other.count[i][j]
		}
	}

	return nil
}

// CountMinSketchJSON is the JSON struct of CountMinSketch for marshal and unmarshal
type CountMinSketchJSON struct {
	D     uint       `json:"d"`
	W     uint       `json:"w"`
	Count [][]uint64 `json:"count"`
}

// MarshalJSON implements json.Marshaler interface.
// Based on https://github.com/willf/bloom/blob/master/bloom.go
func (s *CountMinSketch) MarshalJSON() ([]byte, error) {
	return json.Marshal(CountMinSketchJSON{s.d, s.w, s.count})
}

// UnmarshalJSON implements json.Unmarshaler interface.
// Based on https://github.com/willf/bloom/blob/master/bloom.go
func (s *CountMinSketch) UnmarshalJSON(data []byte) error {
	var j CountMinSketchJSON
	err := json.Unmarshal(data, &j)
	if err != nil {
		return err
	}
	s.d = j.D
	s.w = j.W
	s.count = j.Count
	s.hasher = fnv.New64()
	return nil
}

// WriteTo writes a binary representation of the CountMinSketch to an i/o stream.
// Based on https://github.com/willf/bloom/blob/master/bloom.go
func (s *CountMinSketch) WriteTo(stream io.Writer) (int64, error) {
	err := binary.Write(stream, binary.BigEndian, uint64(s.d))
	if err != nil {
		return 0, err
	}
	err = binary.Write(stream, binary.BigEndian, uint64(s.w))
	if err != nil {
		return 0, err
	}

	C := make([]uint64, s.w)
	for r := uint(0); r < s.d; r++ {
		for c := uint(0); c < s.w; c++ {
			C[c] = s.count[r][c]
		}
		err = binary.Write(stream, binary.BigEndian, C)
		if err != nil {
			return 0, err
		}
	}
	return int64(2*binary.Size(uint64(0)) + int(s.d)*binary.Size(C)), err
}

// ReadFrom a binary representation of the CountMinSketch from an i/o stream.
// Based on https://github.com/willf/bloom/blob/master/bloom.go
func (s *CountMinSketch) ReadFrom(stream io.Reader) (int64, error) {
	var d, w uint64
	err := binary.Read(stream, binary.BigEndian, &d)
	if err != nil {
		return 0, err
	}
	err = binary.Read(stream, binary.BigEndian, &w)
	if err != nil {
		return 0, err
	}
	s.d = uint(d)
	s.w = uint(w)

	s.count = make([][]uint64, s.d)
	for r := uint(0); r < s.d; r++ {
		s.count[r] = make([]uint64, w)
	}

	C := make([]uint64, s.w)
	for r := uint(0); r < s.d; r++ {
		err = binary.Read(stream, binary.BigEndian, &C)
		if err != nil {
			return 0, err
		}
		for c := uint(0); c < s.w; c++ {
			s.count[r][c] = C[c]
		}
	}
	s.hasher = fnv.New64()
	return int64(2*binary.Size(uint64(0)) + int(s.d)*binary.Size(C)), nil
}

// WriteToFile writes the Count-Min Sketch to file
func (s *CountMinSketch) WriteToFile(file string) (int64, error) {
	fh, err := os.Create(file)
	defer fh.Close()
	if err != nil {
		return 0, err
	}
	size, err := s.WriteTo(fh)
	if err != nil {
		return 0, err
	}
	return size, nil
}

// ReadFromFile reads Count-Min Sketch from file
func (s *CountMinSketch) ReadFromFile(file string) (int64, error) {
	fh, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer fh.Close()

	size, err := s.ReadFrom(fh)
	if err != nil {
		return 0, err
	}
	return size, nil
}

// GobEncode implements gob.GobEncoder interface.
func (s *CountMinSketch) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	_, err := s.WriteTo(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder interface.
func (s *CountMinSketch) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	_, err := s.ReadFrom(buf)
	return err
}
