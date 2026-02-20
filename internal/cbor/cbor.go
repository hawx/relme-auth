// Package cbor implements a simple (i.e. probably not good) CBOR decoder
//
// See https://www.rfc-editor.org/rfc/rfc8949.html
package cbor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

var bnry = binary.BigEndian

type Simple int

func (s Simple) String() string {
	return fmt.Sprintf("Simple(%d)", s)
}

type Tag struct {
	Number  uint
	Content any
}

func (f Tag) String() string {
	return fmt.Sprintf("%d(%s)", f.Number, f.Content)
}

func DecodeBytes(data []byte) (any, error) {
	return Decode(bytes.NewReader(data))
}

func Decode(r io.Reader) (any, error) {
	// initial byte
	ib := make([]byte, 1)
	r.Read(ib)
	// major type
	mt := ib[0] >> 5
	// additional information
	ai := ib[0] & 0x1f

	var val any = int(ai)

	switch ai {
	case 24:
		buf := make([]byte, 1)
		r.Read(buf)
		val = int(buf[0])
	case 25:
		buf := make([]byte, 2)
		r.Read(buf)
		num := bnry.Uint16(buf)

		if mt == 7 {
			val = decodeHalf(buf)
		} else {
			val = int(num)
		}
	case 26:
		buf := make([]byte, 4)
		r.Read(buf)
		num := bnry.Uint32(buf)

		if mt == 7 {
			val = math.Float32frombits(num)
		} else {
			val = int(num)
		}
	case 27:
		buf := make([]byte, 8)
		r.Read(buf)
		num := bnry.Uint64(buf)

		if mt == 7 {
			val = math.Float64frombits(num)
		} else {
			val = int(num)
		}
	case 28, 29, 30:
		return nil, errors.New("fail")
	case 31:
		// indefinite
	}

	switch mt {
	case 1:
		val = -1 - val.(int)
	case 2:
		if val == 0 {
			val = []byte{}
		} else if val == 31 {
			arr := []byte{}
			for {
				v, err := Decode(r)
				if v == 31 {
					break
				} else if err != nil {
					return nil, err
				}
				arr = append(arr, v.([]byte)...)
			}
			val = arr
		} else {
			buf := make([]byte, val.(int))
			r.Read(buf)
			val = buf
		}
	case 3:
		if val == 0 {
			val = ""
		} else if val == 31 {
			str := ""
			for {
				v, err := Decode(r)
				if v == 31 {
					break
				} else if err != nil {
					return nil, err
				}
				str += v.(string)
			}
			val = str
		} else {
			buf := make([]byte, val.(int))
			r.Read(buf)
			val = string(buf)
		}
	case 4:
		var arr []any
		for range val.(int) {
			v, err := Decode(r)
			if v == 31 {
				break
			} else if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		val = arr
	case 5:
		pairs := map[any]any{}

		for range val.(int) {
			k, err := Decode(r)
			if k == 31 {
				break
			} else if err != nil {
				return nil, err
			}

			v, err := Decode(r)
			if v == 31 {
				break
			} else if err != nil {
				return nil, err
			}

			pairs[k] = v
		}

		val = pairs
	case 6:
		v, err := Decode(r)
		if err != nil {
			return nil, err
		}

		val = Tag{Number: uint(val.(int)), Content: v}
	case 7:
		if ai < 20 {
			val = Simple(val.(int))
		} else {
			switch ai {
			case 20:
				val = false
			case 21:
				val = true
			case 22:
				val = nil
			case 23:
				val = nil // undefined
			case 24:
				val = Simple(val.(int))
			}
		}
	}

	return val, nil
}

// Taken from https://www.rfc-editor.org/rfc/rfc8949.html#name-half-precision
func decodeHalf(c []byte) float64 {
	half := uint((uint(c[0]) << 8) + uint(c[1]))
	exp := int((half >> 10) & 0x1f)
	mant := half & 0x3ff

	var val float64
	if exp == 0 {
		val = math.Ldexp(float64(mant), -24)
	} else if exp != 31 {
		val = math.Ldexp(float64(mant+1024), exp-25)
	} else if mant == 0 {
		val = math.Inf(1)
	} else {
		val = math.NaN()
	}

	if half&0x8000 > 0 {
		return -val
	}

	return val
}
