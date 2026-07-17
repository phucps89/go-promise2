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
| `types.go` | 167 | Promise API, Then, Map, Catch, Finally |
| `pool.go` | 149 | WorkerPool implementation |
| `combinators.go` | 209 | All, Race, AllSettled, Any, Sequence, Pool |
| `errors.go` | 55 | Error types & AggregateError |
| `doc.go` | 144 | Package documentation & examples |

### Test Files
| File | Mô Tả |
|------|-------|
| `promise_test.go` | Test cho Promise, WorkerPool, combinators |
| `generic_types_test.go` | Test với generic type phức tạp (struct, con trỏ, slice, map) |
| `backpressure_test.go` | Test hành vi block/backpressure của `Submit()` |
| `fuzz_test.go` | Fuzz test cho `WorkerPool` và `AggregateError` |

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
}).Map(context.Background(), func(val int) (int, error) {
    return val * 2, nil  // 10
}).Then(context.Background(), func(val int) error {
    fmt.Println("Value:", val)
    return nil
}).Catch(context.Background(), func(err error) (int, error) {
    return 0, nil
}).Finally(context.Background(), func() {
    fmt.Println("Done!")
})
```

## 📚 API Reference

### Promise Methods
- `NewPromise(fn)` - Tạo promise từ function (panic trong `fn` được tự động recover)
- `NewPromiseWithExecutor(executor)` - Tạo với executor pattern (panic trong `executor` cũng được recover)
- `Await(ctx)` - Chờ kết quả (blocking, gọi được nhiều lần - kết quả được cache lại sau lần resolve đầu tiên)
- `Then(ctx, fn)` - Chuỗi execution
- `Map(ctx, fn)` - Transform value
- `Catch(ctx, fn)` - Xử lý lỗi
- `Finally(ctx, fn)` - Cleanup

### WorkerPool Methods
- `NewWorkerPool[T](numWorkers)` - Tạo pool (`numWorkers <= 0` tự động fallback về 1)
- `Submit(fn)` - Gửi task, trả về Promise ngay. Block nếu queue đã đầy (backpressure) cho tới khi có chỗ trống hoặc pool đóng
- `Close()` - Đóng pool, chờ task đang chạy/còn trong queue chạy xong. An toàn khi gọi đồng thời với `Submit()`, gọi nhiều lần cũng an toàn
- `Stats()` - Lấy thống kê (`NumWorkers`, `QueueSize`, `QueueCapacity`)

### Combinators
- `All(ctx, promises...)` - Tất cả hoàn thành
- `Race(ctx, promises...)` - Nhanh nhất hoàn thành
- `AllSettled(ctx, promises...)` - Tất cả settle
- `Any(ctx, promises...)` - Thành công đầu tiên
- `Sequence(ctx, promises...)` - Chạy tuần tự
- `Pool(ctx, pool, tasks...)` - Chạy trong pool

## 🧪 Tests

40 test cases + 2 fuzz target đều pass (khuyến khích chạy kèm `-race` vì package này khai thác concurrency nhiều):

```bash
go test ./... -race -v

# Chạy fuzz thật (không chỉ seed corpus) trong 15 giây
go test -run '^$' -fuzz FuzzWorkerPoolRoundTrip -fuzztime 15s .
```

Test Coverage:
- ✓ Promise creation, error, executor pattern
- ✓ Promise chaining (Then, Map, Catch, Finally) - kể cả với `ctx` bị hủy
- ✓ `Await()` gọi nhiều lần và đồng thời từ nhiều goroutine vẫn ra đúng kết quả cache
- ✓ Panic trong task/executor được recover, không crash process
- ✓ WorkerPool basic operations, error handling, `Stats()`
- ✓ WorkerPool: `Close()` an toàn khi chạy đồng thời với `Submit()` (race-tested), idempotent, chờ đúng task đang chạy dở
- ✓ WorkerPool: task đã nhận vào queue không bị bỏ dở khi `Close()` được gọi
- ✓ WorkerPool: `Submit()` block đúng khi queue đầy (backpressure) và không làm `Close()` bị treo theo
- ✓ Generic type phức tạp (struct/con trỏ/slice/map) qua Promise và WorkerPool
- ✓ All, Race, AllSettled, Any, Sequence, Pool combinator
- ✓ Context cancellation
- ✓ Fuzz test cho WorkerPool (round-trip kết quả) và AggregateError (format lỗi)

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

- **WorkerPool Overhead**: 1 goroutine cố định cho mỗi worker (không tạo goroutine phụ mỗi lần `Submit()`)
- **Await Latency**: microseconds (blocking operation, an toàn khi gọi nhiều lần vì kết quả được cache)
- **Submit Latency**: nhanh (gần như tức thời) khi queue còn chỗ trống; **block** (backpressure) khi queue đầy cho tới khi có chỗ hoặc pool đóng - không phải non-blocking

## 🛠️ File Locations

```
go-promise2/                           # Module root (github.com/phucps89/go-promise2)
├── types.go                           # Core Promise types
├── pool.go                            # WorkerPool
├── combinators.go                     # Promise combinators
├── errors.go                          # Error types
├── doc.go                             # Package documentation & examples
├── promise_test.go                    # Test chính cho Promise/WorkerPool/combinators
├── generic_types_test.go              # Test với generic type phức tạp
├── backpressure_test.go               # Test Submit() block khi queue đầy
├── fuzz_test.go                       # Fuzz test
├── README.md                          # User guide
├── OVERVIEW.md                        # File này
├── STRUCTURE.md                       # Mô tả cấu trúc & thiết kế
└── go.mod                             # Go module definition
```

## 🎓 How to Use This Package

1. **Đầu tiên**, đọc `README.md` để hiểu API và ví dụ sử dụng
2. **Sau đó**, xem `doc.go` (package doc, hiện trong `go doc`) để có thêm ví dụ
3. **Tiếp theo**, xem `STRUCTURE.md` để hiểu thiết kế nội bộ
4. **Cuối cùng**, `go get github.com/phucps89/go-promise2` và dùng trong project của bạn

## 📝 Notes

- Yêu cầu Go 1.21+ (xem `go.mod`)
- Promise cache kết quả sau khi resolve lần đầu (qua `sync.Once` + channel đóng một lần), nên `Await()`/`Then()`/`Map()`/`Catch()`/`Finally()` gọi nhiều lần hoặc đồng thời từ nhiều goroutine đều an toàn và ra cùng một kết quả
- Panic trong `fn` (của `NewPromise`), trong `executor` (của `NewPromiseWithExecutor`), và trong task chạy qua `WorkerPool` đều được tự động `recover()`, trả về `ErrTaskPanicked` thay vì crash process
- `WorkerPool.Close()` an toàn khi gọi đồng thời với `Submit()` (không còn panic "send on closed channel"), có thể gọi nhiều lần, và đảm bảo task đã được `Submit()` chấp nhận sẽ luôn được chạy trước khi pool đóng hẳn
- `Then`/`Map`/`Catch`/`Finally` nhận `ctx` làm tham số đầu tiên - hủy `ctx` sẽ dừng chờ ngay thay vì treo vô thời hạn nếu promise gốc không bao giờ resolve
- Test suite kèm `-race` và fuzz test cho các đường concurrency quan trọng
- **Giới hạn đã biết**: `Submit()` có khe hở race cực hẹp khi đua với `Close()` (task có thể lọt vào queue đúng lúc pool vừa đóng xong, không còn worker xử lý) khiến `Promise` đó treo vô thời hạn nếu không dùng `ctx` timeout. Xác suất cực thấp, không panic/crash, và là đánh đổi có chủ đích để tránh một deadlock nghiêm trọng hơn khi bịt kín khe hở này bằng mutex. Xem chi tiết ở comment của `Submit()` trong `pool.go` và mục "Giới Hạn Đã Biết" trong README.md. Luôn `Await()` bằng `ctx` có timeout để không bao giờ bị ảnh hưởng bởi khe hở này

---

**License:** MIT — xem lịch sử commit trên GitHub để biết các thay đổi gần nhất
