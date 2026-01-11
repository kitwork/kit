package kit

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"time"
	"unsafe"
)

/* =============================================================================
   1. CORE TYPE DEFINITIONS
   ============================================================================= */

// Kind represents the underlying data type discriminator.
type Kind uint8

const (
	Invalid Kind = iota // Internal error or uninitialized
	Nil                 // Null / undefined

	// --- Scalar Types (Fast-path, data stored in N) ---
	Number   // float64
	Bool     // 0/1 in N
	Time     // UnixNano stored in N
	Duration // Nanoseconds stored in N

	// --- Reference Types (Slow-path, data stored in V) ---
	String // string
	Bytes  // []byte
	Map    // map[string]Value
	Array  // []Value

	// --- Complex Types ---
	Struct // Go struct or pointer
	Func   // Callable function / pipe
	Any    // Opaque Go interface{}
)

// Value is the atomic runtime unit of the engine.
// 24 bytes on 64-bit for cache & stack efficiency.
type Value struct {
	N float64 // Scalar storage
	V any     // Reference storage
	K Kind    // Type discriminator
}

/* =============================================================================
   2. TYPE PREDICATES
   ============================================================================= */

func (v Value) IsInvalid() bool   { return v.K == Invalid }
func (v Value) IsNil() bool       { return v.K == Nil }
func (v Value) IsBlank() bool     { return v.K <= Nil }
func (v Value) IsValid() bool     { return v.K >= Number }
func (v Value) IsImmediate() bool { return v.K <= Duration }

func (v Value) IsScalar() bool { return v.K >= Number && v.K <= Duration }
func (v Value) IsNumeric() bool {
	switch v.K {
	case Number, Time, Duration:
		return true
	default:
		return false
	}
}

func (v Value) IsBool() bool      { return v.K == Bool }
func (v Value) IsTrue() bool      { return v.K == Bool && v.N > 0 }
func (v Value) IsString() bool    { return v.K == String }
func (v Value) IsBytes() bool     { return v.K == Bytes }
func (v Value) IsArray() bool     { return v.K == Array }
func (v Value) IsMap() bool       { return v.K == Map }
func (v Value) IsCallable() bool  { return v.K == Func }
func (v Value) IsReference() bool { return v.K >= String }
func (v Value) IsObject() bool    { return v.K >= String && v.V != nil }

func (v Value) IsIterable() bool {
	switch v.K {
	case Array, Map, Bytes:
		return true
	default:
		return false
	}
}

// Truthy evaluates logical truthiness:
// - Scalars: N > 0
// - Objects: non-nil
func (v Value) Truthy() bool {
	if v.IsImmediate() {
		return v.N > 0
	}
	return v.IsObject()
}

/* =============================================================================
   3. STRINGIFY & CONVERSION
   ============================================================================= */

func (v Value) Text() string {
	buf := make([]byte, 0, 64)
	return string(v.Append(buf))
}

func (v Value) Append(b []byte) []byte {
	switch v.K {
	case String:
		return append(b, v.String()...)
	case Number:
		i := int64(v.N)
		if v.N == float64(i) {
			return strconv.AppendInt(b, i, 10)
		}
		return strconv.AppendFloat(b, v.N, 'g', -1, 64)
	case Bool:
		if v.N > 0 {
			return append(b, "true"...)
		}
		return append(b, "false"...)
	case Nil:
		return append(b, "null"...)
	case Time:
		return time.Unix(0, int64(v.N)).AppendFormat(b, time.RFC3339)
	case Duration:
		return append(b, time.Duration(int64(v.N)).String()...)
	case Bytes:
		return append(b, v.Bytes()...)
	default:
		return b
	}
}

func (v Value) String() string {
	if s, ok := v.V.(string); ok {
		return s
	}
	return ""
}

func (v Value) Int() int64     { return int64(v.N) }
func (v Value) Float() float64 { return v.N }

func (v Value) Bytes() []byte {
	if b, ok := v.V.([]byte); ok {
		return b
	}
	return nil
}

// AsBytes provides a zero-copy read-only view into string data.
func (v Value) AsBytes() []byte {
	if v.K == Bytes {
		return v.Bytes()
	}
	s := v.String()
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func (v Value) ByteSlice() []byte {
	switch v.K {
	case Bytes:
		return v.Bytes()
	case String:
		return v.AsBytes()
	default:
		return nil
	}
}

/* =============================================================================
   4. ARITHMETIC & COMPARISON
   ============================================================================= */

func (a Value) Add(b Value) Value {
	if a.K == Number && b.K == Number {
		return Value{K: Number, N: a.N + b.N}
	}
	return a.Extend(b)
}

func (a Value) Extend(b Value) Value {
	switch {
	case a.K == String || b.K == String:
		return Value{K: String, V: a.Text() + b.Text()}
	case a.K == Time && b.K == Duration:
		return Value{K: Time, N: a.N + b.N}
	default:
		return Value{K: Invalid}
	}
}

func (a Value) Sub(b Value) Value {
	if a.K == Number && b.K == Number {
		return Value{K: Number, N: a.N - b.N}
	}
	if a.K == Time && b.K == Duration {
		return Value{K: Time, N: a.N - b.N}
	}
	return Value{K: Invalid}
}

func (a Value) Mul(b Value) Value {
	if a.K == Number && b.K == Number {
		return Value{K: Number, N: a.N * b.N}
	}
	return Value{K: Invalid}
}

func (a Value) Div(b Value) Value {
	if a.K == Number && b.K == Number {
		if b.N == 0 {
			return Value{K: Nil}
		}
		return Value{K: Number, N: a.N / b.N}
	}
	return Value{K: Invalid}
}

// Deep equality
func (a Value) Equal(b Value) bool {
	if a.K != b.K {
		return false
	}
	switch a.K {
	case Number, Bool, Time, Duration:
		return a.N == b.N
	case String:
		return a.String() == b.String()
	case Nil:
		return true
	case Bytes:
		return bytes.Equal(a.Bytes(), b.Bytes())
	case Array:
		x, y := a.V.([]Value), b.V.([]Value)
		if len(x) != len(y) {
			return false
		}
		for i := range x {
			if !x[i].Equal(y[i]) {
				return false
			}
		}
		return true
	case Map:
		x, y := a.V.(map[string]Value), b.V.(map[string]Value)
		if len(x) != len(y) {
			return false
		}
		for k, xv := range x {
			yv, ok := y[k]
			if !ok || !xv.Equal(yv) {
				return false
			}
		}
		return true
	default:
		return a.V == b.V
	}
}

func (a Value) Less(b Value) bool {
	if a.K <= Duration && b.K <= Duration {
		return a.N < b.N
	}
	if a.K == String && b.K == String {
		return a.String() < b.String()
	}
	return false
}

func (a Value) NotEqual(b Value) bool     { return !a.Equal(b) }
func (a Value) Greater(b Value) bool      { return b.Less(a) }
func (a Value) LessEqual(b Value) bool    { return !b.Less(a) }
func (a Value) GreaterEqual(b Value) bool { return !a.Less(b) }

/* =============================================================================
   5. NAVIGATION & REFLECTION
   ============================================================================= */

func (v Value) Len() int {
	if !v.IsObject() {
		return 0
	}
	switch v.K {
	case String:
		return len(v.V.(string))
	case Bytes:
		return len(v.V.([]byte))
	case Array:
		return len(v.V.([]Value))
	case Map:
		return len(v.V.(map[string]Value))
	}
	return 0
}

func (v Value) Index(i int) Value {
	if !v.IsObject() {
		return Value{K: Nil}
	}
	switch v.K {
	case Array:
		a := v.V.([]Value)
		if i >= 0 && i < len(a) {
			return a[i]
		}
	case Bytes:
		b := v.V.([]byte)
		if i >= 0 && i < len(b) {
			return Value{K: Number, N: float64(b[i])}
		}
	case String:
		s := v.V.(string)
		if i >= 0 && i < len(s) {
			return Value{K: String, V: string(s[i])}
		}
	}
	return Value{K: Nil}
}

func (v Value) Get(key string) Value {
	if !v.IsObject() {
		return Value{K: Nil}
	}
	switch v.K {
	case Map:
		if val, ok := v.V.(map[string]Value)[key]; ok {
			return val
		}
	case Struct:
		return v.reflect(key)
	}
	return Value{K: Nil}
}

// At allows deep path traversal
func (v Value) At(path ...any) Value {
	cur := v
	for _, p := range path {
		switch x := p.(type) {
		case string:
			cur = cur.Get(x)
		case int:
			cur = cur.Index(x)
		default:
			return Value{K: Nil}
		}
		if cur.IsBlank() {
			return cur
		}
	}
	return cur
}

func (v Value) reflect(key string) Value {
	rv := reflect.ValueOf(v.V)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return Value{K: Nil}
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return Value{K: Nil}
	}
	f := rv.FieldByName(key)
	if !f.IsValid() {
		return Value{K: Nil}
	}
	return New(f.Interface())
}

/* =============================================================================
   6. CONSTRUCTORS & NORMALIZATION
   ============================================================================= */

func New(i any) Value {
	if i == nil {
		return Value{K: Nil}
	}

	switch v := i.(type) {
	case Value:
		return v
	case string:
		return Value{K: String, V: v}
	case []byte:
		return Value{K: Bytes, V: v}
	case bool:
		if v {
			return Value{K: Bool, N: 1}
		}
		return Value{K: Bool}
	case int:
		return Value{K: Number, N: float64(v)}
	case float64:
		return Value{K: Number, N: v}
	case time.Time:
		return Value{K: Time, N: float64(v.UnixNano())}
	case time.Duration:
		return Value{K: Duration, N: float64(v.Nanoseconds())}
	case []Value:
		return Value{K: Array, V: v}
	case map[string]Value:
		return Value{K: Map, V: v}
	default:
		return Parse(i)
	}
}

func Parse(i any) Value {
	rv := reflect.ValueOf(i)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return Value{K: Nil}
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return Value{K: Bytes, V: rv.Bytes()}
		}
		n := rv.Len()
		out := make([]Value, n)
		for i := 0; i < n; i++ {
			out[i] = New(rv.Index(i).Interface())
		}
		return Value{K: Array, V: out}

	case reflect.Map:
		out := make(map[string]Value)
		iter := rv.MapRange()
		for iter.Next() {
			var key string
			rk := iter.Key()
			// PERFORMANCE OPTIMIZATION: Avoid fmt.Sprint for string keys
			if rk.Kind() == reflect.String {
				key = rk.String()
			} else {
				key = fmt.Sprint(rk.Interface())
			}
			out[key] = New(iter.Value().Interface())
		}
		return Value{K: Map, V: out}

	case reflect.Struct:
		return Value{K: Struct, V: i}

	default:
		if rv.CanFloat() {
			return Value{K: Number, N: rv.Float()}
		}
		if rv.CanInt() {
			return Value{K: Number, N: float64(rv.Int())}
		}
		return Value{K: Any, V: i}
	}
}
