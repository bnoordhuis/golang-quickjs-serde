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
	"bytes"
	"reflect"
	"testing"
)

func TestReadValue(t *testing.T) {
	if nil != tryReadValue([]byte{bcVersion, 0, 1}) {
		panic("nil expected")
	}
	if Undefined != tryReadValue([]byte{bcVersion, 0, 2}) {
		panic("undefined expected")
	}
	if false != tryReadValue([]byte{bcVersion, 0, 3}) {
		panic("false expected")
	}
	if true != tryReadValue([]byte{bcVersion, 0, 4}) {
		panic("true expected")
	}
	if int32(42) != tryReadValue([]byte{bcVersion, 0, 5, 84}) {
		panic("42 expected")
	}
	if float64(13.37) != tryReadValue([]byte{bcVersion, 0, 6, 61, 10, 215, 163, 112, 189, 42, 64}) {
		panic("13.37 expected")
	}
	// note: quickjs never produces 0.0 but writes a zero int32 instead
	if float64(0.0) != tryReadValue([]byte{bcVersion, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0}) {
		panic("0.0 expected")
	}
	if float64(-0.0) != tryReadValue([]byte{bcVersion, 0, 6, 0, 0, 0, 0, 0, 0, 0, 128}) {
		panic("-0.0 expected")
	}
	if "ok" != tryReadValue([]byte{bcVersion, 0, 7, 4, 111, 107}) {
		panic("\"ok\" expected")
	}
	if "😭" != tryReadValue([]byte{bcVersion, 0, 7, 5, 61, 216, 45, 222}) {
		panic("\"😭\" expected")
	}
	if !reflect.DeepEqual([]any{}, tryReadValue([]byte{bcVersion, 0, 9, 0})) {
		panic("[null] expected")
	}
	if !reflect.DeepEqual([]any{nil}, tryReadValue([]byte{bcVersion, 0, 9, 1, 1})) {
		panic("[null] expected")
	}
	if !reflect.DeepEqual([]byte{}, tryReadValue([]byte{bcVersion, 0, 15, 0})) {
		panic("ArrayBuffer() expected")
	}
	if !reflect.DeepEqual([]byte{42}, tryReadValue([]byte{bcVersion, 0, 15, 1, 42})) {
		panic("ArrayBuffer() expected")
	}
	if !reflect.DeepEqual([]byte{42}, tryReadValue([]byte{bcVersion, 0, 14, 2, 1, 0, 15, 1, 42})) {
		panic("Uint8Array([42]) expected")
	}
	if !reflect.DeepEqual(map[string]any{"k": nil}, tryReadValue([]byte{bcVersion, 1, 2, 107, 8, 1, 2, 1})) {
		panic("{k:null} expected")
	}
	if !reflect.DeepEqual(map[string]any{"42": nil}, tryReadValue([]byte{bcVersion, 0, 8, 1, 85, 1})) {
		panic("{[42]:null} expected")
	}
	if !reflect.DeepEqual(map[string]any{"-42": nil}, tryReadValue([]byte{bcVersion, 1, 6, 45, 52, 50, 8, 1, 2, 1})) {
		panic("{[-42]:null} expected")
	}
}

func TestReadObject(t *testing.T) {
	type empty struct{}
	if !reflect.DeepEqual(&empty{}, tryReadObject(&empty{}, []byte{bcVersion, 0, 8, 0})) {
		panic("empty struct expected")
	}
	if !reflect.DeepEqual(&empty{}, tryReadObject(&empty{}, []byte{bcVersion, 1, 2, 107, 8, 1, 2, 1})) {
		panic("empty struct expected")
	}
}

func tryReadValue(b []byte) any {
	v, err := ReadValue(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	return v
}

func tryReadObject(v any, b []byte) any {
	br := bytes.NewReader(b)
	if err := ReadObject(br, v); err != nil {
		panic(err)
	}
	return v
}
