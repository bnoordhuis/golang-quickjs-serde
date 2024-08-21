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
	"fmt"
	"reflect"
	"testing"
)

func TestReadValue(t *testing.T) {
	expect(nil, tryReadValue([]byte{bcVersion, 0, 1}))
	expect(Undefined, tryReadValue([]byte{bcVersion, 0, 2}))
	expect(false, tryReadValue([]byte{bcVersion, 0, 3}))
	expect(true, tryReadValue([]byte{bcVersion, 0, 4}))
	expect(int32(42), tryReadValue([]byte{bcVersion, 0, 5, 84}))
	expect(float64(13.37), tryReadValue([]byte{bcVersion, 0, 6, 61, 10, 215, 163, 112, 189, 42, 64}))
	// note: quickjs never produces 0.0 but writes a zero int32 instead
	expect(float64(0.0), tryReadValue([]byte{bcVersion, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0}))
	expect(float64(-0.0), tryReadValue([]byte{bcVersion, 0, 6, 0, 0, 0, 0, 0, 0, 0, 128}))
	expect("ok", tryReadValue([]byte{bcVersion, 0, 7, 4, 111, 107}))
	expect("😭", tryReadValue([]byte{bcVersion, 0, 7, 5, 61, 216, 45, 222}))
	expect([]any{}, tryReadValue([]byte{bcVersion, 0, 9, 0}))
	expect([]any{nil}, tryReadValue([]byte{bcVersion, 0, 9, 1, 1}))
	expect([]byte{}, tryReadValue([]byte{bcVersion, 0, 15, 0}))
	expect([]byte{42}, tryReadValue([]byte{bcVersion, 0, 15, 1, 42}))
	expect([]byte{42}, tryReadValue([]byte{bcVersion, 0, 14, 2, 1, 0, 15, 1, 42}))
	expect(map[string]any{"k": nil}, tryReadValue([]byte{bcVersion, 1, 2, 107, 8, 1, 2, 1}))
	expect(map[string]any{"42": nil}, tryReadValue([]byte{bcVersion, 0, 8, 1, 85, 1}))
	expect(map[string]any{"-42": nil}, tryReadValue([]byte{bcVersion, 1, 6, 45, 52, 50, 8, 1, 2, 1}))
}

func TestReadObject(t *testing.T) {
	type empty struct{}
	expect(&empty{}, tryReadObject(&empty{}, []byte{bcVersion, 0, 8, 0}))
	expect(&empty{}, tryReadObject(&empty{}, []byte{bcVersion, 1, 2, 107, 8, 1, 2, 1}))
	// null is coerced to 0
	expect(&struct{ k int }{0}, tryReadObject(&struct{ k int }{42}, []byte{bcVersion, 1, 2, 107, 8, 1, 2, 1}))
	k := 42
	expect(&struct{ k *int }{}, tryReadObject(&struct{ k *int }{&k}, []byte{bcVersion, 1, 2, 107, 8, 1, 2, 1}))
}

func TestWriteValue(t *testing.T) {
	expect([]byte{bcVersion, 0, tagNull}, tryWriteValue(nil))
	expect([]byte{bcVersion, 0, tagUndefined}, tryWriteValue(Undefined))
	expect([]byte{bcVersion, 0, tagTrue}, tryWriteValue(true))
	expect([]byte{bcVersion, 0, tagFalse}, tryWriteValue(false))
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

func tryWriteValue(v any) []byte {
	buf := bytes.Buffer{}
	if err := WriteValue(&buf, v); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func expect(expected any, actual any) {
	if !reflect.DeepEqual(expected, actual) {
		panic(fmt.Sprintf("expected %v, have %v", expected, actual))
	}
}
