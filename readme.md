# Kit – Core Runtime Value Engine

**Kit** is a lightweight, high-performance atomic value system for Go, designed as the **core runtime engine** for expression evaluation, data normalization, and dynamic computation inside larger systems.

It provides a **compact, deterministic representation** of dynamic data while minimizing allocations and reflection overhead. Kit is **not a framework**—it’s a **foundational building block** for any system requiring a consistent, high-speed value runtime.

## ✨ Core Concept: Tagged Value

At the heart of Kit is the `Value` struct, carefully engineered to occupy **24 bytes** on 64-bit architectures for **cache-friendly and stack-efficient** performance.

### Fast Path – Scalars

Primitive types are stored **directly inside the struct**:

* `Number` (float64)
* `Bool` (0/1 encoded)
* `Time` (Unix nanoseconds)
* `Duration` (nanoseconds)

Benefits:

* No heap allocations
* Minimal GC pressure
* Optimized for hot execution paths

### Slow Path – References

Complex types are stored as **references**:

* `String`
* `Bytes`
* `Array` (`[]Value`)
* `Map` (`map[string]Value`)
* `Struct` (Go struct or pointer)

This design guarantees **predictable memory behavior** while supporting rich and nested data structures.

## 🚀 Key Features

* **Unified Entry Point** – Use `kit.New(i any)` to normalize any Go data into a `Value`, including slices, maps, and structs.
* **Reflection Only Once** – Reflection is used **only during parsing**; all runtime operations are reflection-free.
* **Zero-Copy String Access** – Read-only string data as `[]byte` via `AsBytes()` with no additional allocation.
* **Deep Navigation** – Fluent and safe access with `Get()`, `Index()`, and `At()`.
* **Immutable Semantics** – Values are effectively immutable after creation, safe for concurrent reads.

## 🧰 API Overview

```go
import "github.com/kitwork/kit"

// Normalize arbitrary Go data
v := kit.New(map[string]any{
    "id": 101,
    "meta": []string{"admin", "active"},
})

// Fluent access
id := v.Get("id").Int()          // 101
status := v.At("meta", 1).Text() // "active"

// Safe comparison
if v.Get("id").Greater(kit.New(100)) {
    // Perform logic...
}
```

## 📊 Performance Highlights

Kit is designed for **high-frequency evaluation**:

| Operation                           | Approx. Cost    |
| ----------------------------------- | --------------- |
| Scalar arithmetic                   | ~2–3 ns/op      |
| Reference access (string/map/array) | ~5–7 ns/op      |
| Deep parse (complex data)           | Input-dependent |

Because Kit values are immutable and reflection-free after creation, they are **thread-safe and highly cache-friendly**, ideal for engine-level computations.

## 📑 Design Decisions

### 1. Fixed 24-byte Layout

* **Why**: Avoid `interface{}` overhead, improve cache locality, and allow compiler inlining.
* **Impact**: Values are small, predictable, and stack-allocated whenever possible.

### 2. Reflection Only at Parse Time

* **Why**: Reflection is slow; limiting it to parsing ensures predictable hot-path performance.
* **Impact**: Runtime operations are fast and safe.

### 3. Immutable Values

* **Why**: Simplifies reasoning, makes values thread-safe, and fits engine-style evaluation.
* **Impact**: Deep operations (e.g., `At()`) are safe under concurrent usage.

## 📝 Summary

Kit is a **compact, deterministic, and high-performance value runtime**, ideal for any Go system requiring:

* Consistent representation of dynamic data
* Safe deep navigation and manipulation
* Minimal heap allocation and GC pressure
* Fast, predictable operations

It forms the **core engine** for larger runtime systems, expression evaluators, and rule engines.

## 📄 License

Released under the MIT License.

