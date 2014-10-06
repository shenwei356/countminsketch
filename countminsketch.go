/*
countmin - An implementation of Count-Min Sketch in Golang.

http://github.com/shenwei356/countmin/

The code is deeply inspired by an implementation of Bloom filters in golang,
[bloom](https://github.com/willf/bloom).
*/

package countminsketch

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"math"
	"os"
)

// CountMinSketch struct. d is the number of hashing functions,
// The hash function maps items uniformly onto the range {1, 2, . . . w}
// count store the count data. slice, instead of matrix, is used
// for convenience in serialization operation.
type CountMinSketch struct {
	d      uint
	w      uint
	count  []uint
	hasher hash.Hash64
}

// Create a new Count-Min Sketch with _d_ hashing functions
// and _w_ hash value range
func New(d uint, w uint) (s *CountMinSketch, err error) {
	// the cap of slice is int, int32 exacly.
	if int32(d*w) < 0 || int32(d*w) > int32(^uint32(0)>>1) {
		return nil, fmt.Errorf("d*w (%d) too large, out of range", d*w)
	}

	// assumming you have a 2G RAM, the cap of slice should < (2<<30) / 4
	if int32(d*w) > 536870912 {
		return nil, fmt.Errorf("d*w (%d) too large, it would be out of memory", d*w)
	}
	s = &CountMinSketch{
		d:      d,
		w:      w,
		count:  make([]uint, int32(d*w)),
		hasher: fnv.New64(),
	}
	return s, err
}

// Create a new Count-Min Sketch with given error rate and confidence.
// Accuracy guarantees will be made in terms of a pair of user specified parameters,
// ε and δ, meaning that the error in answering a query is within a factor of ε with
// probability δ
func NewWithEstimates(varepsilon, delta float64) (*CountMinSketch, error) {
	if delta >= 1.0 {
		delta = 0.9999
	}
	w := uint(math.Ceil(2 / varepsilon))
	d := uint(math.Ceil(math.Log(1-delta) / math.Log(0.5)))
	return New(d, w)
}

// Create a new Count-Min Sketch from dumpped file
func NewFromFile(file string) (*CountMinSketch, error) {
	s, _ := New(0, 0)
	_, err := s.ReadFromFile(file)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Return the number of hashing functions
func (s *CountMinSketch) D() uint {
	return s.d
}

// Return the w
func (s *CountMinSketch) W() uint {
	return s.w
}

// get the two basic hash function values for data.
// Based on https://github.com/willf/bloom/blob/master/bloom.go
func (s *CountMinSketch) base_hashes(key []byte) (a uint32, b uint32) {
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
	a, b := s.base_hashes(key)
	ua := uint(a)
	ub := uint(b)
	for r := uint(0); r < s.d; r++ {
		locs[r] = (ua + ub*r) % s.w
	}
	return
}

// Update the frequency of a key
func (s *CountMinSketch) Update(key []byte, count uint) {
	for r, c := range s.locations(key) {
		s.count[uint(r)*s.w+c] += count
	}
}

// Update the frequency of a key
func (s *CountMinSketch) UpdateString(key string, count uint) {
	s.Update([]byte(key), count)
}

// Estimate the frequency of a key. It is point query.
func (s *CountMinSketch) Estimate(key []byte) uint {
	var min uint
	for r, c := range s.locations(key) {
		if r == 0 || s.count[uint(r)*s.w+c] < min {
			min = s.count[uint(r)*s.w+c]
		}
	}
	return min
}

// Estimate the frequency of a key
func (s *CountMinSketch) EstimateString(key string) uint {
	return s.Estimate([]byte(key))
}

// JSON struct for marshal and unmarshal
type countMinSketchJSON struct {
	D     uint   `json:"d"`
	W     uint   `json:"w"`
	Count []uint `json:"count"`
}

// MarshalJSON implements json.Marshaler interface.
// Based on https://github.com/willf/bloom/blob/master/bloom.go
func (s *CountMinSketch) MarshalJSON() ([]byte, error) {
	return json.Marshal(countMinSketchJSON{s.d, s.w, s.count})
}

// UnmarshalJSON implements json.Unmarshaler interface.
// Based on https://github.com/willf/bloom/blob/master/bloom.go
func (s *CountMinSketch) UnmarshalJSON(data []byte) error {
	var j countMinSketchJSON
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

	C := make([]uint64, s.d*s.w)
	for i := uint(0); i < s.d*s.w; i++ {
		C[i] = uint64(s.count[i])
	}
	err = binary.Write(stream, binary.BigEndian, C)
	if err != nil {
		return 0, err
	}
	return int64(2*binary.Size(uint64(0)) + binary.Size(C)), err
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
	s.count = make([]uint, s.d*s.w)

	C := make([]uint64, s.d*s.w)
	err = binary.Read(stream, binary.BigEndian, &C)
	if err != nil {
		return 0, err
	}
	for i := uint(0); i < s.d*s.w; i++ {
		s.count[i] = uint(C[i])
	}

	s.hasher = fnv.New64()
	return int64(2*binary.Size(uint64(0)) + binary.Size(C)), nil
}

// Write the Count-Min Sketch to file
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

// Read Count-Min Sketch from file
func (s *CountMinSketch) ReadFromFile(file string) (int64, error) {
	fh, err := os.Open(file)
	defer fh.Close()
	if err != nil {
		return 0, err
	}
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
