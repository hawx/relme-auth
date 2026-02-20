package cbor

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"testing"

	"hawx.me/code/assert"
)

func TestDecode(t *testing.T) {
	// https://www.rfc-editor.org/rfc/rfc8949.html#name-examples-of-encoded-cbor-da
	testcases := []struct {
		in  string
		out any
	}{
		{"00", 0},
		{"01", 1},
		{"0a", 10},
		{"17", 23},
		{"1818", 24},
		{"1819", 25},
		{"1864", 100},
		{"1903e8", 1000},
		{"1a000f4240", 1000000},
		{"1b000000e8d4a51000", 1000000000000},
		// {"1bffffffffffffffff", 18446744073709551615},
		// {"c249010000000000000000", 18446744073709551616},
		// {"3bffffffffffffffff", -18446744073709551616},
		// {"c349010000000000000000", -18446744073709551617},
		{"20", -1},
		{"29", -10},
		{"3863", -100},
		{"3903e7", -1000},
		{"f90000", 0.0},
		{"f98000", -0.0},
		{"f93c00", 1.0},
		{"fb3ff199999999999a", 1.1},
		{"f93e00", 1.5},
		{"f97bff", 65504.0},
		{"fa47c35000", float32(100000.0)},
		{"fa7f7fffff", float32(3.4028234663852886e+38)},
		{"fb7e37e43c8800759c", 1.0e+300},
		{"f90001", 5.960464477539063e-8},
		{"f90400", 0.00006103515625},
		{"f9c400", -4.0},
		{"fbc010666666666666", -4.1},
		{"f4", false},
		{"f5", true},
		{"f6", nil},
		{"f7", nil}, // undefined
		{"f0", Simple(16)},
		{"f8ff", Simple(255)},
		{"c074323031332d30332d32315432303a30343a30305a", Tag{Number: 0, Content: "2013-03-21T20:04:00Z"}},
		{"c11a514b67b0", Tag{Number: 1, Content: 1363896240}},
		{"c1fb41d452d9ec200000", Tag{Number: 1, Content: 1363896240.5}},
		{"d74401020304", Tag{Number: 23, Content: []byte{0x01, 0x02, 0x03, 0x04}}},
		{"d818456449455446", Tag{Number: 24, Content: []byte{0x64, 0x49, 0x45, 0x54, 0x46}}},
		{"d82076687474703a2f2f7777772e6578616d706c652e636f6d", Tag{Number: 32, Content: "http://www.example.com"}},
		{"40", []byte{}},
		{"4401020304", []byte{1, 2, 3, 4}},
		{"60", ""},
		{"6161", "a"},
		{"6449455446", "IETF"},
		{"62225c", "\"\\"},
		{"62c3bc", "\u00fc"},
		{"63e6b0b4", "\u6c34"},
		// {"64f0908591", "\ud800\udd51"},
		{"80", []any(nil)},
		{"83010203", []any{1, 2, 3}},
		{"8301820203820405", []any{1, []any{2, 3}, []any{4, 5}}},
		{"98190102030405060708090a0b0c0d0e0f101112131415161718181819", []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}},
		{"a0", map[any]any{}},
		{"a201020304", map[any]any{1: 2, 3: 4}},
		{"a26161016162820203", map[any]any{"a": 1, "b": []any{2, 3}}},
		{"826161a161626163", []any{"a", map[any]any{"b": "c"}}},
		{"a56161614161626142616361436164614461656145", map[any]any{"a": "A", "b": "B", "c": "C", "d": "D", "e": "E"}},
		{"5f42010243030405ff", []byte{1, 2, 3, 4, 5}},
		{"7f657374726561646d696e67ff", "streaming"},
		{"9fff", []any(nil)},
		{"9f018202039f0405ffff", []any{1, []any{2, 3}, []any{4, 5}}},
		{"9f01820203820405ff", []any{1, []any{2, 3}, []any{4, 5}}},
		{"83018202039f0405ff", []any{1, []any{2, 3}, []any{4, 5}}},
		{"83019f0203ff820405", []any{1, []any{2, 3}, []any{4, 5}}},
		{"9f0102030405060708090a0b0c0d0e0f101112131415161718181819ff", []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}},
		{"bf61610161629f0203ffff", map[any]any{"a": 1, "b": []any{2, 3}}},
		{"826161bf61626163ff", []any{"a", map[any]any{"b": "c"}}},
		{"bf6346756ef563416d7421ff", map[any]any{"Fun": true, "Amt": -2}},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("%v is %v", tc.in, tc.out), func(t *testing.T) {
			v, _ := hex.DecodeString(tc.in)
			result, err := DecodeBytes(v)

			if assert.Nil(t, err) {
				assert.Equal(t, tc.out, result)
			}
		})
	}
}

func TestDecodeWebauthn(t *testing.T) {
	bs, _ := base64.RawURLEncoding.DecodeString("o2NmbXRkbm9uZWdhdHRTdG10oGhhdXRoRGF0YViUSZYN5YgOjGh0NBcPZHZgW4_krrmihjLHmVzzuoMdl2NdAAAAAAAAAAAAAAAAAAAAAAAAAAAAEFd9lVZ3sLfMfDXI1HIGlPylAQIDJiABIVggSyvwsYanK0zitcQIS6Ccth0lG7zY_Cwe8PhggXo2j_MiWCB4aB8EhCcOb6S1cdC_sV4P4cVwVJ-dLVa6eYWop7k91g")

	v, err := DecodeBytes(bs)

	assert.Nil(t, err)
	assert.NotNil(t, v)

	authData := v.(map[any]any)["authData"].([]byte)

	idLen := binary.BigEndian.Uint16(authData[53:55])
	credentialID := authData[55 : 55+idLen]
	publicKeyBytes := authData[55+idLen:]

	publicKeyObject, err := DecodeBytes(publicKeyBytes)

	assert.Nil(t, err)
	assert.Equal(t, []byte{0x57, 0x7d, 0x95, 0x56, 0x77, 0xb0, 0xb7, 0xcc, 0x7c, 0x35, 0xc8, 0xd4, 0x72, 0x6, 0x94, 0xfc}, credentialID)
	assert.Nil(t, publicKeyObject)
	// then store (userID, credentialID, publicKeyBytes) for authenticating
	// https://webauthn.guide/#registration
}

func TestDecodeFloats(t *testing.T) {
	t.Run("float16 +Inf", func(t *testing.T) {
		v, _ := hex.DecodeString("f97c00")
		result, err := DecodeBytes(v)
		if err != nil {
			t.Log("got error:", err.Error())
			t.Fail()
		} else if !math.IsInf(result.(float64), 1) {
			t.Logf("expected %v == Inf", result)
			t.Fail()
		}
	})

	t.Run("float16 NaN", func(t *testing.T) {
		v, _ := hex.DecodeString("f97e00")
		result, err := DecodeBytes(v)
		if err != nil {
			t.Log("got error:", err.Error())
			t.Fail()
		} else if !math.IsNaN(result.(float64)) {
			t.Logf("expected %v == NaN", result)
			t.Fail()
		}
	})

	t.Run("float16 -Inf", func(t *testing.T) {
		v, _ := hex.DecodeString("f9fc00")
		result, err := DecodeBytes(v)
		if err != nil {
			t.Log("got error:", err.Error())
			t.Fail()
		} else if !math.IsInf(result.(float64), -1) {
			t.Logf("expected %v == -Inf", result)
			t.Fail()
		}
	})

	t.Run("float32 +Inf", func(t *testing.T) {
		v, _ := hex.DecodeString("fa7f800000")
		result, err := DecodeBytes(v)
		if err != nil {
			t.Log("got error:", err.Error())
			t.Fail()
		} else if !math.IsInf(float64(result.(float32)), 1) {
			t.Logf("expected %v == Inf", result)
			t.Fail()
		}
	})

	t.Run("float32 NaN", func(t *testing.T) {
		v, _ := hex.DecodeString("fa7fc00000")
		result, err := DecodeBytes(v)
		if err != nil {
			t.Log("got error:", err.Error())
			t.Fail()
		} else if !math.IsNaN(float64(result.(float32))) {
			t.Logf("expected %v == NaN", result)
			t.Fail()
		}
	})

	t.Run("float32 -Inf", func(t *testing.T) {
		v, _ := hex.DecodeString("faff800000")
		result, err := DecodeBytes(v)
		if err != nil {
			t.Log("got error:", err.Error())
			t.Fail()
		} else if !math.IsInf(float64(result.(float32)), -1) {
			t.Logf("expected %v == -Inf", result)
			t.Fail()
		}
	})

	t.Run("float64 +Inf", func(t *testing.T) {
		v, _ := hex.DecodeString("fb7ff0000000000000")
		result, err := DecodeBytes(v)
		if err != nil {
			t.Log("got error:", err.Error())
			t.Fail()
		} else if !math.IsInf(result.(float64), 1) {
			t.Logf("expected %v == Inf", result)
			t.Fail()
		}
	})

	t.Run("float64 NaN", func(t *testing.T) {
		v, _ := hex.DecodeString("fb7ff8000000000000")
		result, err := DecodeBytes(v)
		if err != nil {
			t.Log("got error:", err.Error())
			t.Fail()
		} else if !math.IsNaN(result.(float64)) {
			t.Logf("expected %v == NaN", result)
			t.Fail()
		}
	})

	t.Run("float64 NaN", func(t *testing.T) {
		v, _ := hex.DecodeString("fbfff0000000000000")
		result, err := DecodeBytes(v)
		if err != nil {
			t.Log("got error:", err.Error())
			t.Fail()
		} else if !math.IsInf(result.(float64), -1) {
			t.Logf("expected %v == -Inf", result)
			t.Fail()
		}
	})
}
