// SPDX-License-Identifier: 0BSD
// Package mpack is a minimal MessagePack decoder sufficient for
// Reticulum storage files (maps, arrays, bin, str, floats, ints, nil).
// Stdlib only.
package mpack

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Decode reads one MessagePack value from r.
func Decode(r io.Reader) (any, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return nil, err
	}
	return decode(r, b[0])
}

func readN(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

func readInt(r io.Reader, n int, signed bool) (int64, error) {
	buf, err := readN(r, n)
	if err != nil {
		return 0, err
	}
	switch n {
	case 1:
		if signed {
			return int64(int8(buf[0])), nil // #nosec G115 -- bounded msgpack header field, length-checked
		}
		return int64(buf[0]), nil
	case 2:
		v := binary.BigEndian.Uint16(buf)
		if signed {
			return int64(int16(v)), nil // #nosec G115 -- bounded msgpack header field, length-checked
		}
		return int64(v), nil
	case 4:
		v := binary.BigEndian.Uint32(buf)
		if signed {
			return int64(int32(v)), nil // #nosec G115 -- bounded msgpack header field, length-checked
		}
		return int64(v), nil
	case 8:
		v := binary.BigEndian.Uint64(buf)
		if signed {
			return int64(v), nil // #nosec G115 -- bounded msgpack header field, length-checked
		}
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("uint64 overflow")
		}
		return int64(v), nil
	}
	return 0, fmt.Errorf("bad int size %d", n)
}

func decode(r io.Reader, marker byte) (any, error) {
	switch {
	case marker <= 0x7f:
		return int64(marker), nil
	case marker >= 0x80 && marker <= 0x8f: // fixmap
		return decodeMap(r, int(marker&0x0f))
	case marker >= 0x90 && marker <= 0x9f: // fixarray
		return decodeArray(r, int(marker&0x0f))
	case marker >= 0xa0 && marker <= 0xbf: // fixstr
		buf, err := readN(r, int(marker&0x1f))
		return string(buf), err
	case marker >= 0xe0: // negative fixint
		return int64(int8(marker)), nil // #nosec G115 -- bounded msgpack header field, length-checked
	}
	switch marker {
	case 0xc0:
		return nil, nil
	case 0xc2:
		return false, nil
	case 0xc3:
		return true, nil
	case 0xc4, 0xc5, 0xc6: // bin8/16/32
		n, err := readInt(r, 1<<(marker-0xc4), false)
		if err != nil {
			return nil, err
		}
		if n > 64<<20 {
			return nil, fmt.Errorf("bin too large")
		}
		return readN(r, int(n))
	case 0xca: // float32
		buf, err := readN(r, 4)
		return float64(math.Float32frombits(binary.BigEndian.Uint32(buf))), err
	case 0xcb: // float64
		buf, err := readN(r, 8)
		return math.Float64frombits(binary.BigEndian.Uint64(buf)), err
	case 0xcc, 0xcd, 0xce, 0xcf: // uint8..64
		return readInt(r, 1<<(marker-0xcc), false)
	case 0xd0, 0xd1, 0xd2, 0xd3: // int8..64
		return readInt(r, 1<<(marker-0xd0), true)
	case 0xd9, 0xda, 0xdb: // str8/16/32
		n, err := readInt(r, 1<<(marker-0xd9), false)
		if err != nil {
			return nil, err
		}
		buf, err := readN(r, int(n))
		return string(buf), err
	case 0xdc, 0xdd: // array16/32
		n, err := readInt(r, 2<<(marker-0xdc), false)
		if err != nil {
			return nil, err
		}
		return decodeArray(r, int(n))
	case 0xde, 0xdf: // map16/32
		n, err := readInt(r, 2<<(marker-0xde), false)
		if err != nil {
			return nil, err
		}
		return decodeMap(r, int(n))
	}
	return nil, fmt.Errorf("unsupported marker 0x%02x", marker)
}

func decodeMap(r io.Reader, n int) (map[any]any, error) {
	m := make(map[any]any, n)
	for range n {
		k, err := Decode(r)
		if err != nil {
			return nil, err
		}
		v, err := Decode(r)
		if err != nil {
			return nil, err
		}
		if kb, ok := k.([]byte); ok {
			k = string(kb) // slices are unhashable; string(b) keeps raw bytes
		}
		m[k] = v
	}
	return m, nil
}

func decodeArray(r io.Reader, n int) ([]any, error) {
	a := make([]any, 0, n)
	for range n {
		v, err := Decode(r)
		if err != nil {
			return nil, err
		}
		a = append(a, v)
	}
	return a, nil
}
