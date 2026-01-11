package kit

import (
	"testing"
)

// --- UNIT TESTS ---

func TestValue_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		wantKind Kind
		wantN    float64
	}{
		{"Number int", 42, Number, 42},
		{"Number float", 3.14, Number, 3.14},
		{"Bool true", true, Bool, 1},
		{"Bool false", false, Bool, 0},
		{"Nil", nil, Nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New(tt.input)
			if v.K != tt.wantKind {
				t.Errorf("New() Kind = %v, want %v", v.K, tt.wantKind)
			}
			if v.IsImmediate() && v.N != tt.wantN {
				t.Errorf("New() N = %v, want %v", v.N, tt.wantN)
			}
		})
	}
}

func TestValue_DeepParsing(t *testing.T) {
	// Test mảng lồng nhau: [][]int
	input := [][]int{{1}, {2, 3}}
	v := New(input)

	if v.K != Array {
		t.Fatal("Expected Array kind for nested slice")
	}

	// Kiểm tra phần tử [1][1] (số 3)
	val := v.Index(1).Index(1)
	if val.N != 3 {
		t.Errorf("Deep parsing failed: expected 3, got %v", val.N)
	}
}

func TestValue_StructReflection(t *testing.T) {
	type User struct {
		Name string
		Age  int
	}
	u := User{Name: "Kitwork", Age: 1}
	v := New(u)

	if v.K != Struct {
		t.Fatal("Expected Struct kind")
	}

	// Test Get field từ Struct thông qua Reflection
	name := v.Get("Name")
	if name.String() != "Kitwork" {
		t.Errorf("Struct reflection failed: expected 'Kitwork', got '%s'", name.String())
	}
}

func TestValue_ZeroCopyBytes(t *testing.T) {
	s := "hello world"
	v := New(s)

	b := v.AsBytes()
	if string(b) != s {
		t.Errorf("AsBytes failed: expected '%s', got '%s'", s, string(b))
	}
}

// --- BENCHMARKS ---

func BenchmarkNew_Number(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New(1234.56)
	}
}

func BenchmarkNew_String(b *testing.B) {
	s := "performance test"
	for i := 0; i < b.N; i++ {
		_ = New(s)
	}
}

func BenchmarkNew_SliceComplex(b *testing.B) {
	// Đo tốc độ xử lý một mảng phức tạp cần Reflection
	input := []map[string]any{
		{"id": 1, "ok": true},
		{"id": 2, "ok": false},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New(input)
	}
}
