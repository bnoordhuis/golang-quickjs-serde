// Copyright (c) 2024, Ben Noordhuis <info@bnoordhuis.nl>
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package serde

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"unicode/utf16"
)

const bcVersion = 12

// corresponds with BCTagEnum in quickjs.c
const (
	tagNull = 1 + iota
	tagUndefined
	tagFalse
	tagTrue
	tagInt32
	tagFloat64
	tagString
	tagObject
	tagArray
	tagBigInt
	tagTemplateObject
	tagFunctionBytecode
	tagModule
	tagTypedArray
	tagArrayBuffer
	tagSharedArrayBuffer
	tagRegExp
	tagDate
	tagObjectValue
	tagObjectReference
)

type UndefinedValue struct{}

var Undefined = UndefinedValue{}

func ReadObject(r io.Reader) (v any, err error) {
	defer func() {
		if x := recover(); x != nil {
			switch v := x.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("serde.ReadObject: %v", v)
			}
		}
	}()
	br := bufio.NewReader(r)
	if version := readByte(br); version != bcVersion {
		panic(fmt.Sprintf("version mismatch (have %d, want %d)", version, bcVersion))
	}
	atomCount := int(readUint32(br))
	atoms := make([]string, atomCount)
	for i := 0; i < atomCount; i++ {
		atoms[i] = readString(br)
	}
	v = readObject(br, atoms)
	return
}

func WriteObject(w io.Writer, v any) error {
	return nil
}

func readObject(r *bufio.Reader, atoms []string) any {
	switch tag := readByte(r); tag {
	case tagNull:
		return nil
	case tagUndefined:
		return Undefined
	case tagFalse:
		return false
	case tagTrue:
		return true
	case tagInt32:
		v, err := binary.ReadVarint(r)
		if err != nil {
			panic(err)
		}
		if v < math.MinInt32 || v > math.MaxInt32 {
			panic(fmt.Sprintf("int32 out of range: %d", v))
		}
		return int32(v)
	case tagFloat64:
		var v float64
		panicIf(binary.Read(r, binary.LittleEndian, &v))
		return v
	case tagString:
		return readString(r)
	case tagObject:
		n := int(readUint32(r))
		m := make(map[string]any, n)
		for i := 0; i < n; i++ {
			k := int(readUint32(r))
			isTaggedInt := (k & 1) == 1
			k = k >> 1
			var atom string
			if isTaggedInt {
				atom = fmt.Sprintf("%d", k)
			} else if k > 0 && k <= len(atoms) {
				atom = atoms[k-1]
			} else {
				panic("atom out of range")
			}
			m[atom] = readObject(r, atoms)
		}
		return m
	case tagArray:
		n := readUint32(r)
		v := make([]any, n)
		for i := uint32(0); i < n; i++ {
			v[i] = readObject(r, atoms)
		}
		return v
	case tagArrayBuffer:
		n := readUint32(r)
		return readBytes(r, n)
	case tagTypedArray:
		tag := readByte(r)
		if tag > 10 {
			panic(fmt.Sprintf("bad typed array tag: %d", tag))
		}
		n := readUint32(r)
		// offset into arraybuffer (t time of serialization;
		// *not* an offset into the arraybuffer following
		// this typed array
		_ = readUint32(r)
		if tagArrayBuffer != readByte(r) {
			panic("typed array not followed by arraybuffer")
		}
		if n != readUint32(r) {
			panic("typed array not followed by arraybuffer of right size")
		}
		switch tag {
		case 0: // Uint8ClampedArray
			v := make([]byte, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 2: // Uint8Array
			v := make([]byte, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 1: // Int8Array
			v := make([]int8, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 3: // Int16Array
			v := make([]int16, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 4: // Uint16Array
			v := make([]uint16, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 5: // Int32Array
			v := make([]int32, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 6: // Uint32Array
			v := make([]uint32, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 7: // BigInt64Array
			v := make([]int64, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 8: // BigUint64Array
			v := make([]uint64, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 9: // Float32Array
			v := make([]float32, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case 10: // Float64Array
			v := make([]float64, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		default:
			panic("unreachable")
		}
	case tagFunctionBytecode:
		panic("bytecode not supported")
	case tagModule:
		panic("module not supported")
	default:
		panic(fmt.Sprintf("unknown tag %02x", tag))
	}
}

func readByte(r *bufio.Reader) byte {
	if b, err := r.ReadByte(); err == nil {
		return b
	} else {
		panic(err)
	}
}

func readBytes(r *bufio.Reader, n uint32) []byte {
	b := make([]byte, n)
	if _, err := r.Read(b); err != nil {
		panic(err)
	}
	return b
}

func readUint32(r *bufio.Reader) uint32 {
	v, err := binary.ReadUvarint(r)
	if err != nil {
		panic(err)
	}
	if v > math.MaxUint32 {
		panic(fmt.Sprintf("uint32 out of range: %d", v))
	}
	return uint32(v)
}

func readString(r *bufio.Reader) string {
	n := readUint32(r)
	isWide := (n & 1) == 1
	n = n >> 1
	if isWide {
		h := make([]uint16, n)
		panicIf(binary.Read(r, binary.LittleEndian, &h))
		return string(utf16.Decode(h))
	} else {
		b := readBytes(r, n)
		return string(b)
	}
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
