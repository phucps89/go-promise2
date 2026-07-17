# Promise2 - Promise Pattern for Go with Worker Pool

`promise2` là một package Go cung cấp Promise pattern (giống JavaScript) kết hợp với Worker Pool để xử lý concurrent tasks một cách đơn giản và hiệu quả.

## Cài Đặt

```bash
go get github.com/phucps89/go-promise2
```

## Tính Năng

✅ **Promise API** - Tương tự JavaScript Promise  
✅ **Worker Pool** - Quản lý goroutines hiệu quả, có backpressure  
✅ **Combinators** - All, Race, AllSettled, Any, Sequence, Pool  
✅ **Chainable** - Then, Map, Catch, Finally  
✅ **Context Support** - Hỗ trợ cancellation xuyên suốt các API chain  
✅ **Panic Safety** - Panic trong task/executor được tự động recover, không crash process  
✅ **Repeated Await** - `Await()`/`Then()`/... gọi nhiều lần hoặc đồng thời đều an toàn, kết quả được cache  

## Cách Sử Dụng

### 1. Promise Cơ Bản

```go
package main

import (
	"context"
	"fmt"
	"github.com/phucps89/go-promise2"
)

func main() {
	promise := promise2.NewPromise(func() (string, error) {
		return "Hello World", nil
	})

	result, err := promise.Await(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(result) // "Hello World"
}
```

### 2. Promise với Executor

```go
promise := promise2.NewPromiseWithExecutor[string](func(resolve func(string), reject func(error)) {
	// Thực hiện công việc bất đồng bộ
	result, err := someAsyncOperation()
	if err != nil {
		reject(err)
		return
	}
	resolve(result)
})

result, err := promise.Await(context.Background())
```

### 3. Promise Chaining - Then

```go
promise := promise2.NewPromise(func() (int, error) {
	return 10, nil
}).Then(context.Background(), func(val int) error {
	fmt.Println("Received:", val)
	return nil
})
```

### 4. Transform - Map

```go
promise := promise2.NewPromise(func() (int, error) {
	return 5, nil
}).Map(context.Background(), func(val int) (int, error) {
	return val * 2, nil  // Kết quả: 10
})
```

### 5. Error Handling - Catch

```go
promise := promise2.NewPromise(func() (int, error) {
	return 0, fmt.Errorf("something failed")
}).Catch(context.Background(), func(err error) (int, error) {
	fmt.Println("Caught error:", err)
	return 0, nil  // Recover with default value
})
```

### 6. Cleanup - Finally

```go
promise := promise2.NewPromise(func() (string, error) {
	return "done", nil
}).Finally(context.Background(), func() {
	fmt.Println("Cleanup")
})
```

> `Then`/`Map`/`Catch`/`Finally` đều nhận `ctx` làm tham số đầu tiên: nếu `ctx`
> bị hủy trước khi promise gốc resolve, bước chain đó sẽ dừng chờ ngay với
> `ctx.Err()` thay vì treo vô thời hạn.

### 7. Worker Pool

```go
// Tạo pool với 4 workers
pool := promise2.NewWorkerPool[int](4)
defer pool.Close()

// Submit tasks
p1 := pool.Submit(func() (int, error) {
	// Long running task
	return 42, nil
})

p2 := pool.Submit(func() (int, error) {
	return 100, nil
})

// Chờ kết quả
r1, _ := p1.Await(context.Background())
r2, _ := p2.Await(context.Background())
fmt.Println(r1, r2) // 42 100
```

> `Submit()` block (backpressure) nếu hàng đợi đã đầy, cho tới khi có chỗ
> trống hoặc pool bị đóng - không tạo goroutine phụ cho mỗi task. Panic bên
> trong task được tự động recover và trả về `ErrTaskPanicked` thay vì làm
> crash worker. `Close()` an toàn khi gọi đồng thời với `Submit()` từ
> goroutine khác, và có thể gọi nhiều lần.

### 8. Promise.All() - Chờ tất cả hoàn thành

```go
p1 := promise2.NewPromise(func() (string, error) { return "task1", nil })
p2 := promise2.NewPromise(func() (string, error) { return "task2", nil })
p3 := promise2.NewPromise(func() (string, error) { return "task3", nil })

results, err := promise2.All(context.Background(), p1, p2, p3).Await(context.Background())
if err != nil {
	panic(err)
}
// results: [task1, task2, task3]
```

Nếu bất kỳ promise nào lỗi, `All()` sẽ reject ngay:

```go
p1 := promise2.NewPromise(func() (int, error) { return 1, nil })
p2 := promise2.NewPromise(func() (int, error) { return 0, fmt.Errorf("failed") })

_, err := promise2.All(context.Background(), p1, p2).Await(context.Background())
// err != nil
```

### 9. Promise.Race() - Chờ cái nhanh nhất

```go
p1 := promise2.NewPromise(func() (int, error) {
	time.Sleep(2 * time.Second)
	return 1, nil
})

p2 := promise2.NewPromise(func() (int, error) {
	return 2, nil  // Nhanh hơn
})

winner, _ := promise2.Race(context.Background(), p1, p2).Await(context.Background())
fmt.Println(winner) // 2
```

### 10. Promise.AllSettled() - Chờ tất cả settle (dù thành công hay thất bại)

```go
p1 := promise2.NewPromise(func() (int, error) { return 1, nil })
p2 := promise2.NewPromise(func() (int, error) { return 0, fmt.Errorf("error") })

results, _ := promise2.AllSettled(context.Background(), p1, p2).Await(context.Background())

for i, status := range results {
	if status.Status == promise2.StatusFulfilled {
		fmt.Printf("[%d] Success: %v\n", i, status.Value)
	} else {
		fmt.Printf("[%d] Error: %v\n", i, status.Err)
	}
}
// [0] Success: 1
// [1] Error: error
```

### 11. Promise.Any() - Chờ cái thành công đầu tiên

```go
p1 := promise2.NewPromise(func() (int, error) { return 0, fmt.Errorf("fail1") })
p2 := promise2.NewPromise(func() (int, error) { return 42, nil })
p3 := promise2.NewPromise(func() (int, error) { return 0, fmt.Errorf("fail3") })

result, _ := promise2.Any(context.Background(), p1, p2, p3).Await(context.Background())
fmt.Println(result) // 42

// Nếu tất cả fail:
p1 := promise2.NewPromise(func() (int, error) { return 0, fmt.Errorf("fail1") })
p2 := promise2.NewPromise(func() (int, error) { return 0, fmt.Errorf("fail2") })

_, err := promise2.Any(context.Background(), p1, p2).Await(context.Background())
if ae, ok := err.(*promise2.AggregateError); ok {
	fmt.Printf("All %d promises failed\n", ae.Count())
	for i, e := range ae.Errors() {
		fmt.Printf("  [%d] %v\n", i, e)
	}
}
```

### 12. Promise.Sequence() - Chạy tuần tự

```go
p1 := promise2.NewPromise(func() (int, error) { return 1, nil })
p2 := promise2.NewPromise(func() (int, error) { return 2, nil })
p3 := promise2.NewPromise(func() (int, error) { return 3, nil })

results, _ := promise2.Sequence(context.Background(), p1, p2, p3).Await(context.Background())
// results: [1, 2, 3]
```

### 13. Pool Helper - Chạy tasks trong worker pool

```go
pool := promise2.NewWorkerPool[int](4)
defer pool.Close()

tasks := []func() (int, error){
	func() (int, error) { return 1, nil },
	func() (int, error) { return 2, nil },
	func() (int, error) { return 3, nil },
}

results, _ := promise2.Pool(context.Background(), pool, tasks...).Await(context.Background())
// results: [1, 2, 3]
```

### 14. Worker Pool Statistics

```go
pool := promise2.NewWorkerPool[int](4)
defer pool.Close()

// Lấy thống kê
stats := pool.Stats()
fmt.Printf("Workers: %d\n", stats.NumWorkers)
fmt.Printf("Queue Size: %d\n", stats.QueueSize)
fmt.Printf("Queue Capacity: %d\n", stats.QueueCapacity)
```

### 15. Xử lý lỗi đặc biệt

Package định nghĩa sẵn 3 loại lỗi để nhận diện bằng `errors.Is`/type assertion:

```go
result, err := promise.Await(ctx)
switch {
case errors.Is(err, promise2.ErrTaskPanicked):
	// task hoặc executor panic, đã được recover
case errors.Is(err, promise2.ErrPoolClosed):
	// Submit() vào một WorkerPool đã Close()
case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
	// ctx truyền vào Await()/Then()/... bị hủy trước khi promise gốc resolve
default:
	if ae, ok := err.(*promise2.AggregateError); ok {
		// tất cả promises trong Any() đều bị reject
		fmt.Println("all failed:", ae.Count())
	}
}
```

## API Reference

### Promise[T]

| Method | Mô Tả |
|--------|-------|
| `NewPromise(fn)` | Tạo promise từ function. Panic trong `fn` được recover, trả về `ErrTaskPanicked` |
| `NewPromiseWithExecutor(executor)` | Tạo promise với executor pattern. Panic trong `executor` cũng được recover |
| `Await(ctx)` | Chờ kết quả (blocking). Gọi được nhiều lần, kể cả đồng thời từ nhiều goroutine - kết quả được cache sau lần resolve đầu tiên |
| `Then(ctx, fn)` | Chuỗi thực thi sau promise hoàn thành. Hủy `ctx` sẽ dừng chờ ngay với `ctx.Err()` |
| `Map(ctx, fn)` | Transform giá trị của promise |
| `Catch(ctx, fn)` | Xử lý lỗi (kể cả lỗi do `ctx` hết hạn khi chờ promise gốc) |
| `Finally(ctx, fn)` | Cleanup - luôn chạy dù thành công, lỗi, hay `ctx` hết hạn |

### WorkerPool[T]

| Method | Mô Tả |
|--------|-------|
| `NewWorkerPool(numWorkers)` | Tạo worker pool. `numWorkers <= 0` tự động fallback về 1 |
| `Submit(fn)` | Gửi task vào pool, trả về Promise ngay. **Block** nếu queue đã đầy (backpressure) cho tới khi có chỗ hoặc pool đóng |
| `Close()` | Đóng pool, chờ task đang chạy/còn trong queue chạy xong. An toàn khi gọi đồng thời với `Submit()`, gọi nhiều lần cũng an toàn (idempotent) |
| `Stats()` | Lấy thống kê về pool (`NumWorkers`, `QueueSize`, `QueueCapacity`) |

### Combinators

| Function | Mô Tả |
|----------|-------|
| `All(ctx, promises...)` | Chờ tất cả promises hoàn thành |
| `Race(ctx, promises...)` | Chờ promise hoàn thành đầu tiên |
| `AllSettled(ctx, promises...)` | Chờ tất cả promises settle |
| `Any(ctx, promises...)` | Chờ promise success đầu tiên |
| `Sequence(ctx, promises...)` | Chạy promises theo thứ tự |
| `Pool(ctx, pool, tasks...)` | Chạy tasks trong worker pool |

## ⚠️ Giới Hạn Đã Biết

`WorkerPool.Submit()` có một khe hở race **cực hẹp, xác suất cực thấp**: nếu
`Submit()` và `Close()` đua nhau đúng lúc (`Submit()` vừa thấy pool chưa đóng
thì `Close()` lập tức chạy xong hoàn toàn, kể cả chờ hết mọi worker thoát),
task có thể vẫn lọt được vào queue dù không còn worker nào xử lý - `Promise`
đó sẽ treo vô thời hạn ở `Await()` nếu không có `ctx` timeout. Đây không phải
panic hay crash, và hầu hết implementation worker pool đơn giản đều có giới
hạn tương tự (bịt hoàn toàn khe hở này bằng mutex sẽ đổi lại bằng nguy cơ
`Close()` bị deadlock nếu có `Submit()` đang chờ chỗ trống trong lúc worker
kẹt ở một task không bao giờ trả về - rủi ro nghiêm trọng hơn nhiều).

**Khuyến nghị**: luôn `Await()` bằng `ctx` có timeout, đặc biệt với các task
được `Submit()` gần thời điểm pool có thể bị `Close()`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
result, err := promise.Await(ctx)
```

## Best Practices

1. **Luôn Close Worker Pool**
   ```go
   pool := promise2.NewWorkerPool[int](4)
   defer pool.Close()
   ```

2. **Sử dụng Context để Cancel**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   result, _ := promise.Await(ctx)
   ```

3. **Proper Error Handling**
   ```go
   result, err := promise.Await(ctx)
   if err != nil {
       // Handle error
   }
   ```

4. **Worker Pool Size**
   - Đặt = số CPU cores cho CPU-bound tasks
   - Đặt cao hơn (2-4x) cho I/O-bound tasks

## Ví Dụ Thực Tế

### Xử lý Multiple HTTP Requests

```go
pool := promise2.NewWorkerPool[string](10)
defer pool.Close()

urls := []string{"url1", "url2", "url3"}
promises := make([]*promise2.Promise[string], len(urls))

for i, url := range urls {
	promises[i] = pool.Submit(func() (string, error) {
		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		return string(body), err
	})
}

results, err := promise2.All(context.Background(), promises...).Await(context.Background())
```

### Database Batch Operations

```go
pool := promise2.NewWorkerPool[bool](5)
defer pool.Close()

items := []Item{...}
promises := make([]*promise2.Promise[bool], len(items))

for i, item := range items {
	promises[i] = pool.Submit(func() (bool, error) {
		// Database insert operation
		err := db.Save(item)
		return err == nil, err
	})
}

_, err := promise2.All(context.Background(), promises...).Await(context.Background())
```

## Chạy Tests

```bash
# Chạy toàn bộ test, khuyến khích kèm -race vì package khai thác concurrency nhiều
go test ./... -race -v

# Fuzz thật (không chỉ chạy seed corpus) cho WorkerPool trong 15 giây
go test -run '^$' -fuzz FuzzWorkerPoolRoundTrip -fuzztime 15s .
```

## License

MIT
