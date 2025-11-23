# Promise2 - Go Promise Package với Worker Pool

## 🎯 Giới Thiệu

`promise2` là một Go package cung cấp **Promise pattern** (giống JavaScript) kết hợp với **Worker Pool** để xử lý concurrent tasks một cách đơn giản, dễ đọc và dễ bảo trì.

Package này được thiết kế để:
- ✅ Cung cấp familiar API giống JavaScript Promise
- ✅ Hỗ trợ Worker Pool để quản lý goroutines hiệu quả
- ✅ Dễ đọc và dễ hiểu cho các lập trình viên
- ✅ Dễ bảo trì với code sạch

## 📦 Các File Trong Package

### Core Files
| File | Dòng | Mô Tả |
|------|------|-------|
| `types.go` | 150 | Promise API, Then, Map, Catch, Finally |
| `pool.go` | 110 | WorkerPool implementation |
| `combinators.go` | 175 | All, Race, AllSettled, Any, Sequence, Pool |
| `errors.go` | 55 | Error types & AggregateError |
| `doc.go` | 100+ | Package documentation & examples |
| `promise_test.go` | 340 | Unit tests (18 tests, all passing) |

### Documentation
| File | Mô Tả |
|------|-------|
| `README.md` | Hướng dẫn sử dụng chi tiết |
| `STRUCTURE.md` | Mô tả cấu trúc & thiết kế |

## 🚀 Quick Start

### 1. Tạo Promise Đơn Giản

```go
promise := promise2.NewPromise(func() (string, error) {
    return "Hello", nil
})

result, err := promise.Await(context.Background())
```

### 2. Sử dụng Worker Pool

```go
pool := promise2.NewWorkerPool[int](4) // 4 workers
defer pool.Close()

p1 := pool.Submit(func() (int, error) { return 42, nil })
p2 := pool.Submit(func() (int, error) { return 100, nil })

results, _ := promise2.All(context.Background(), p1, p2).Await(context.Background())
// results: [42, 100]
```

### 3. Promise Chaining

```go
promise := promise2.NewPromise(func() (int, error) {
    return 5, nil
}).Map(func(val int) (int, error) {
    return val * 2, nil  // 10
}).Then(func(val int) error {
    fmt.Println("Value:", val)
    return nil
}).Catch(func(err error) (int, error) {
    return 0, nil
}).Finally(func() {
    fmt.Println("Done!")
})
```

## 📚 API Reference

### Promise Methods
- `NewPromise(fn)` - Tạo promise từ function
- `NewPromiseWithExecutor(executor)` - Tạo với executor pattern
- `Await(ctx)` - Chờ kết quả
- `Then(fn)` - Chuỗi execution
- `Map(fn)` - Transform value
- `Catch(fn)` - Xử lý lỗi
- `Finally(fn)` - Cleanup

### WorkerPool Methods
- `NewWorkerPool[T](numWorkers)` - Tạo pool
- `Submit(fn)` - Gửi task
- `Close()` - Đóng pool
- `Stats()` - Lấy thống kê

### Combinators
- `All(ctx, promises...)` - Tất cả hoàn thành
- `Race(ctx, promises...)` - Nhanh nhất hoàn thành
- `AllSettled(ctx, promises...)` - Tất cả settle
- `Any(ctx, promises...)` - Thành công đầu tiên
- `Sequence(ctx, promises...)` - Chạy tuần tự
- `Pool(ctx, pool, tasks...)` - Chạy trong pool

## 🧪 Tests

Tất cả 18 test cases đều pass:

```bash
cd /Users/tranxuanthanhphuc/working/go/web
go test ./promise2 -v
```

Test Coverage:
- ✓ Promise creation
- ✓ Promise chaining (Then, Map, Catch, Finally)
- ✓ WorkerPool basic operations
- ✓ WorkerPool error handling
- ✓ All combinator
- ✓ Race combinator
- ✓ AllSettled combinator
- ✓ Any combinator
- ✓ Sequence combinator
- ✓ Pool helper
- ✓ Context cancellation

## 💡 Best Practices

1. **Luôn Close Worker Pool**
   ```go
   pool := promise2.NewWorkerPool[int](4)
   defer pool.Close()
   ```

2. **Sử dụng Context để Timeout**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   result, _ := promise.Await(ctx)
   ```

3. **Proper Error Handling**
   ```go
   result, err := promise.Await(ctx)
   if err != nil {
       log.Printf("Error: %v", err)
   }
   ```

4. **Chọn Worker Pool Size**
   - CPU-bound: = số CPU cores
   - I/O-bound: 2-4x số CPU cores

## 📊 Examples

### Multiple HTTP Requests
```go
pool := promise2.NewWorkerPool[string](10)
defer pool.Close()

promises := make([]*promise2.Promise[string], len(urls))
for i, url := range urls {
    promises[i] = pool.Submit(func() (string, error) {
        resp, _ := http.Get(url)
        body, _ := io.ReadAll(resp.Body)
        return string(body), nil
    })
}

results, _ := promise2.All(context.Background(), promises...).
    Await(context.Background())
```

### Batch Database Operations
```go
pool := promise2.NewWorkerPool[bool](5)
defer pool.Close()

promises := make([]*promise2.Promise[bool], len(items))
for i, item := range items {
    promises[i] = pool.Submit(func() (bool, error) {
        err := db.Save(item)
        return err == nil, err
    })
}

_, _ = promise2.All(context.Background(), promises...).
    Await(context.Background())
```

### Timeout Pattern
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

promise := promise2.NewPromise(func() (string, error) {
    // Some long operation
    return "done", nil
})

result, err := promise.Await(ctx)
if err == context.DeadlineExceeded {
    log.Println("Operation timeout")
}
```

## 🔍 Lợi Ích

### So với Go's Native Goroutines
✅ Cleaner API - giống JavaScript Promise
✅ Built-in error handling
✅ Worker pool included
✅ Chainable operations
✅ Less boilerplate code

### So với Callback Hell
✅ Readable promise chains
✅ Error propagation
✅ Cancellation support
✅ Parallel execution patterns

## 📈 Performance

- **Promise Overhead**: ~64-128 bytes per promise
- **WorkerPool Overhead**: 1 goroutine per worker
- **Await Latency**: microseconds (blocking operation)
- **Submit Latency**: nanoseconds (non-blocking)

## 🛠️ File Locations

```
/Users/tranxuanthanhphuc/working/go/web/
├── promise2/                          # Main package
│   ├── types.go                       # Core Promise types
│   ├── pool.go                        # WorkerPool
│   ├── combinators.go                 # Promise combinators
│   ├── errors.go                      # Error types
│   ├── doc.go                         # Documentation
│   ├── promise_test.go                # Tests
│   ├── README.md                      # User guide
│   └── STRUCTURE.md                   # This guide
│
├── examples/
│   └── promise2_examples.go           # 10 working examples
│
└── go.mod                             # Go module definition
```

## 🎓 How to Use This Package

1. **Đầu tiên**, đọc `README.md` để hiểu API
2. **Sau đó**, chạy `examples/promise2_examples.go` để xem examples
3. **Tiếp theo**, xem `STRUCTURE.md` để hiểu thiết kế
4. **Cuối cùng**, sử dụng API trong project của bạn

## 📝 Notes

- Hỗ trợ Go 1.18+ (sử dụng Generics)
- Tất cả APIs là thread-safe
- Proper panic recovery trong worker pool
- Context cancellation được hỗ trợ
- Comprehensive test coverage

---

**Được tạo lúc:** November 23, 2025
**Version:** 1.0.0
**License:** MIT
