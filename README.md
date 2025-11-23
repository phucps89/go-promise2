# Promise2 - Promise Pattern for Go with Worker Pool

`promise2` là một package Go cung cấp Promise pattern (giống JavaScript) kết hợp với Worker Pool để xử lý concurrent tasks một cách đơn giản và hiệu quả.

## Tính Năng

✅ **Promise API** - Tương tự JavaScript Promise  
✅ **Worker Pool** - Quản lý goroutines hiệu quả  
✅ **Combinators** - All, Race, AllSettled, Any, Sequence  
✅ **Chainable** - Then, Map, Catch, Finally  
✅ **Context Support** - Hỗ trợ cancellation  
✅ **Dễ đọc & dễ bảo trì** - Code sạch và well-documented  

## Cài Đặt

```go
import "path/to/promise2"
```

## Cách Sử Dụng

### 1. Promise Cơ Bản

```go
package main

import (
	"context"
	"fmt"
	"path/to/promise2"
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
}).Then(func(val int) error {
	fmt.Println("Received:", val)
	return nil
})
```

### 4. Transform - Map

```go
promise := promise2.NewPromise(func() (int, error) {
	return 5, nil
}).Map(func(val int) (int, error) {
	return val * 2, nil  // Kết quả: 10
})
```

### 5. Error Handling - Catch

```go
promise := promise2.NewPromise(func() (int, error) {
	return 0, fmt.Errorf("something failed")
}).Catch(func(err error) (int, error) {
	fmt.Println("Caught error:", err)
	return 0, nil  // Recover with default value
})
```

### 6. Cleanup - Finally

```go
promise := promise2.NewPromise(func() (string, error) {
	return "done", nil
}).Finally(func() {
	fmt.Println("Cleanup")
})
```

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

// Lấy thống kê
stats := pool.Stats()
fmt.Printf("Workers: %d\n", stats.NumWorkers)
fmt.Printf("Queue Size: %d\n", stats.QueueSize)
fmt.Printf("Queue Capacity: %d\n", stats.QueueCapacity)
```

## API Reference

### Promise[T]

| Method | Mô Tả |
|--------|-------|
| `NewPromise(fn)` | Tạo promise từ function |
| `NewPromiseWithExecutor(executor)` | Tạo promise với executor pattern |
| `Await(ctx)` | Chờ kết quả (blocking) |
| `Then(fn)` | Chuỗi thực thi sau promise hoàn thành |
| `Map(fn)` | Transform giá trị của promise |
| `Catch(fn)` | Xử lý lỗi |
| `Finally(fn)` | Cleanup - luôn chạy dù success hay fail |

### WorkerPool[T]

| Method | Mô Tả |
|--------|-------|
| `NewWorkerPool(numWorkers)` | Tạo worker pool |
| `Submit(fn)` | Gửi task vào pool, trả về Promise |
| `Close()` | Đóng pool, chờ tất cả tasks hoàn thành |
| `Stats()` | Lấy thống kê về pool |

### Combinators

| Function | Mô Tả |
|----------|-------|
| `All(ctx, promises...)` | Chờ tất cả promises hoàn thành |
| `Race(ctx, promises...)` | Chờ promise hoàn thành đầu tiên |
| `AllSettled(ctx, promises...)` | Chờ tất cả promises settle |
| `Any(ctx, promises...)` | Chờ promise success đầu tiên |
| `Sequence(ctx, promises...)` | Chạy promises theo thứ tự |
| `Pool(ctx, pool, tasks...)` | Chạy tasks trong worker pool |

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
go test ./promise2 -v
```

## License

MIT
