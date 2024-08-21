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
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"unicode/utf16"
	"unsafe"
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

const (
	uint8ClampedArray = iota
	int8Array
	uint8Array
	int16Array
	uint16Array
	int32Array
	uint32Array
	bigInt64Array
	bigUint64Array
	float32Array
	float64Array
)

type ArrayBuffer struct{ Bytes []byte }
type Uint8ClampedArray struct{ Bytes []byte }
type UndefinedValue struct{}

var Undefined = UndefinedValue{}

func ReadValue(r io.Reader) (v any, err error) {
	defer func() {
		if x := recover(); x != nil {
			switch v := x.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("serde.ReadValue: %v", v)
			}
		}
	}()
	atoms := readHeader(r)
	v = readValue(r, atoms)
	return
}

func ReadObject(r io.Reader, v any) (err error) {
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
	atoms := readHeader(r)
	if tag := readByte(r); tag != tagObject {
		panic(fmt.Sprintf("object expected, have %s", tagName(tag)))
	}
	count := readUint32(r) // property count
	for i := 0; i < count; i++ {
		name := readAtom(r, atoms)
		value := readValue(r, atoms)
		setField(v, name, value)
	}
	return nil
}

// The wire format is somewhat inefficient in that object keys ("atoms")
// go at the front, so you have to buffer the output until you're sure
// you've seen all objects.
func WriteValue(w io.Writer, v any) (err error) {
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
	atoms := []string{} // TODO
	write(w, []byte{bcVersion})
	writeUvarint(w, len(atoms))
	switch t := v.(type) {
	case nil:
		write(w, []byte{tagNull})
	case UndefinedValue:
		write(w, []byte{tagUndefined})
	case bool:
		b := byte(tagFalse)
		if t {
			b = tagTrue
		}
		write(w, []byte{b})
	case ArrayBuffer:
		write(w, []byte{tagArrayBuffer})
		writeUvarint(w, len(t.Bytes))
		write(w, t.Bytes)
	case Uint8ClampedArray:
		writeTypedArray(w, len(t.Bytes), t.Bytes, uint8ClampedArray)
	case []byte:
		writeTypedArray(w, len(t), t, uint8Array)
	case []int8:
		writeTypedArray(w, len(t), t, int8Array)
	case []int16:
		writeTypedArray(w, len(t), t, int16Array)
	case []uint16:
		writeTypedArray(w, len(t), t, uint16Array)
	case []int32:
		writeTypedArray(w, len(t), t, int32Array)
	case []uint32:
		writeTypedArray(w, len(t), t, uint32Array)
	case []int64:
		writeTypedArray(w, len(t), t, bigInt64Array)
	case []uint64:
		writeTypedArray(w, len(t), t, bigUint64Array)
	case []float32:
		writeTypedArray(w, len(t), t, float32Array)
	case []float64:
		writeTypedArray(w, len(t), t, float64Array)
	default:
		panic(fmt.Sprintf("unsupported type %t", t))
	}
	return nil
}

func writeTypedArray(w io.Writer, n int, v any, tag byte) {
	write(w, []byte{tagTypedArray, tag})
	writeUvarint(w, n)
	writeUvarint(w, 0)
	write(w, []byte{tagArrayBuffer})
	writeUvarint(w, n)
	panicIf(binary.Write(w, binary.LittleEndian, v))
}

func write(w io.Writer, b []byte) {
	if _, err := w.Write(b); err != nil {
		panic(err)
	}
}

func writeUvarint(w io.Writer, v int) {
	var b [8]byte
	n := binary.PutUvarint(b[:], uint64(v))
	write(w, b[:n])
}

func readHeader(r io.Reader) []string {
	if version := readByte(r); version != bcVersion {
		panic(fmt.Sprintf("version mismatch (have %d, want %d)", version, bcVersion))
	}
	count := readUint32(r)
	atoms := make([]string, count)
	for i := 0; i < count; i++ {
		atoms[i] = readString(r)
	}
	return atoms
}

func readAtom(r io.Reader, atoms []string) string {
	idx := readUint32(r)
	isTaggedInt := (idx & 1) == 1
	idx = idx >> 1
	if isTaggedInt {
		return fmt.Sprintf("%d", idx)
	}
	if idx > 0 && idx <= len(atoms) {
		return atoms[idx-1]
	}
	panic("atom out of range")
}

func readValue(r io.Reader, atoms []string) any {
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
		v, err := binary.ReadVarint(byteReader{r})
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
		n := readUint32(r)
		m := make(map[string]any, n)
		for i := 0; i < n; i++ {
			atom := readAtom(r, atoms)
			m[atom] = readValue(r, atoms)
		}
		return m
	case tagArray:
		n := readUint32(r)
		v := make([]any, n)
		for i := 0; i < n; i++ {
			v[i] = readValue(r, atoms)
		}
		return v
	case tagArrayBuffer:
		n := readUint32(r)
		return readBytes(r, n)
	case tagTypedArray:
		tag := readByte(r)
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
		case uint8ClampedArray:
			v := make([]byte, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case uint8Array:
			v := make([]byte, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case int8Array:
			v := make([]int8, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case int16Array:
			v := make([]int16, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case uint16Array:
			v := make([]uint16, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case int32Array:
			v := make([]int32, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case uint32Array:
			v := make([]uint32, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case bigInt64Array:
			v := make([]int64, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case bigUint64Array:
			v := make([]uint64, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case float32Array:
			v := make([]float32, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		case float64Array:
			v := make([]float64, n)
			panicIf(binary.Read(r, binary.LittleEndian, &v))
			return v
		default:
			panic(fmt.Sprintf("bad typed array tag: %d", tag))
		}
	default:
		panic(fmt.Sprintf("unsupported %s", tagName(tag)))
	}
}

type byteReader struct {
	r io.Reader
}

func (br byteReader) ReadByte() (res byte, err error) {
	var b [1]byte
	_, err = br.r.Read(b[:])
	res = b[0]
	return
}

func readByte(r io.Reader) byte {
	br := byteReader{r}
	if b, err := br.ReadByte(); err == nil {
		return b
	} else {
		panic(err)
	}
}

func readBytes(r io.Reader, n int) []byte {
	b := make([]byte, n)
	if _, err := r.Read(b); err != nil && n > 0 {
		panic(err)
	}
	return b
}

func readUint32(r io.Reader) int {
	v, err := binary.ReadUvarint(byteReader{r})
	if err != nil {
		panic(err)
	}
	if v > math.MaxUint32 || v > math.MaxInt {
		panic(fmt.Sprintf("uint32 out of range: %d", v))
	}
	return int(v)
}

func readString(r io.Reader) string {
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

func setField(ptr any, name string, value any) bool {
	pv := reflect.ValueOf(ptr).Elem()
	field, ok := pv.Type().FieldByName(name)
	if ok {
		fv := pv.FieldByIndex(field.Index)
		fp := unsafe.Pointer(fv.UnsafeAddr())
		fv = reflect.NewAt(fv.Type(), fp).Elem()
		vv := reflect.ValueOf(value)
		if vv.IsValid() {
			fv.Set(vv)
		} else {
			fv.SetZero()
		}
	}
	return ok
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func tagName(tag byte) string {
	switch tag {
	case tagNull:
		return "null"
	case tagUndefined:
		return "undefined"
	case tagFalse:
		return "false"
	case tagTrue:
		return "true"
	case tagInt32:
		return "int32"
	case tagFloat64:
		return "float64"
	case tagString:
		return "string"
	case tagObject:
		return "object"
	case tagArray:
		return "array"
	case tagBigInt:
		return "bigint"
	case tagTemplateObject:
		return "template object"
	case tagFunctionBytecode:
		return "function bytecode"
	case tagModule:
		return "module"
	case tagTypedArray:
		return "typed array" // TODO include type
	case tagArrayBuffer:
		return "arraybuffer"
	case tagSharedArrayBuffer:
		return "sharedarraybuffer"
	case tagRegExp:
		return "regexp"
	case tagDate:
		return "date"
	case tagObjectValue:
		return "object value"
	case tagObjectReference:
		return "object reference"
	}
	return fmt.Sprintf("unknown tag %d", tag)
}
