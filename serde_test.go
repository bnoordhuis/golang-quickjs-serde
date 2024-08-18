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

func TestReadObject(t *testing.T) {
	if nil != tryReadObject([]byte{bcVersion, 0, 1}) {
		panic("nil expected")
	}
	if Undefined != tryReadObject([]byte{bcVersion, 0, 2}) {
		panic("undefined expected")
	}
	if false != tryReadObject([]byte{bcVersion, 0, 3}) {
		panic("false expected")
	}
	if true != tryReadObject([]byte{bcVersion, 0, 4}) {
		panic("true expected")
	}
	if int32(42) != tryReadObject([]byte{bcVersion, 0, 5, 84}) {
		panic("42 expected")
	}
	if float64(13.37) != tryReadObject([]byte{bcVersion, 0, 6, 61, 10, 215, 163, 112, 189, 42, 64}) {
		panic("13.37 expected")
	}
	// note: quickjs never produces 0.0 but writes a zero int32 instead
	if float64(0.0) != tryReadObject([]byte{bcVersion, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0}) {
		panic("0.0 expected")
	}
	if float64(-0.0) != tryReadObject([]byte{bcVersion, 0, 6, 0, 0, 0, 0, 0, 0, 0, 128}) {
		panic("-0.0 expected")
	}
	if "ok" != tryReadObject([]byte{bcVersion, 0, 7, 4, 111, 107}) {
		panic("\"ok\" expected")
	}
	if "😭" != tryReadObject([]byte{bcVersion, 0, 7, 5, 61, 216, 45, 222}) {
		panic("\"😭\" expected")
	}
	if !reflect.DeepEqual([]any{}, tryReadObject([]byte{bcVersion, 0, 9, 0})) {
		panic("[null] expected")
	}
	if !reflect.DeepEqual([]any{nil}, tryReadObject([]byte{bcVersion, 0, 9, 1, 1})) {
		panic("[null] expected")
	}
	if !reflect.DeepEqual([]byte{}, tryReadObject([]byte{bcVersion, 0, 15, 0})) {
		panic("ArrayBuffer() expected")
	}
	if !reflect.DeepEqual([]byte{42}, tryReadObject([]byte{bcVersion, 0, 15, 1, 42})) {
		panic("ArrayBuffer() expected")
	}
	if !reflect.DeepEqual([]byte{42}, tryReadObject([]byte{bcVersion, 0, 14, 2, 1, 0, 15, 1, 42})) {
		panic("Uint8Array([42]) expected")
	}
	if !reflect.DeepEqual(map[string]any{"k": nil}, tryReadObject([]byte{bcVersion, 1, 2, 107, 8, 1, 2, 1})) {
		panic("{k:null} expected")
	}
	if !reflect.DeepEqual(map[string]any{"42": nil}, tryReadObject([]byte{bcVersion, 0, 8, 1, 85, 1})) {
		panic("{[42]:null} expected")
	}
	if !reflect.DeepEqual(map[string]any{"-42": nil}, tryReadObject([]byte{bcVersion, 1, 6, 45, 52, 50, 8, 1, 2, 1})) {
		panic("{[-42]:null} expected")
	}
}

func tryReadObject(b []byte) any {
	v, err := ReadObject(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	return v
}
