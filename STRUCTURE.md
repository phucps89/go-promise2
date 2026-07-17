# Promise2 Package - Cấu Trúc & Thiết Kế

## 📁 Cấu Trúc Thư Mục

```
go-promise2/                    # Module root (github.com/phucps89/go-promise2)
├── types.go                    # Core types: Promise, Result
├── pool.go                     # WorkerPool implementation
├── combinators.go              # Promise combinators: All, Race, Any, etc.
├── errors.go                   # Error definitions: AggregateError, etc.
├── doc.go                      # Package documentation (go doc)
├── promise_test.go             # Unit tests cho Promise/WorkerPool/combinators
├── generic_types_test.go       # Test với generic type phức tạp (struct, con trỏ, slice, map)
├── backpressure_test.go        # Test hành vi block/backpressure của Submit()
├── fuzz_test.go                # Fuzz test cho WorkerPool và AggregateError
├── README.md                   # User guide
├── OVERVIEW.md                 # Tổng quan package
└── STRUCTURE.md                # This file
```

## 📄 Mô Tả Chi Tiết Từng File

### 1. `types.go` - Core Types
**Chứa:**
- `Result[T]` - Struct chứa giá trị hoặc lỗi
- `Promise[T]` - Struct chính đại diện cho async operation (`done chan struct{}` + `result` cache + `resolveOnce sync.Once`)
- `NewPromise(fn)` - Tạo promise từ function; panic trong `fn` được `recover()`, trả về `ErrTaskPanicked`
- `NewPromiseWithExecutor(executor)` - Tạo promise với executor pattern; panic trong `executor` cũng được recover
- `Await(ctx)` - Chờ kết quả của promise. Gọi được nhiều lần, kể cả đồng thời từ nhiều goroutine - kết quả cache lại sau lần resolve đầu
- `Then(ctx, fn)` - Chuỗi promise
- `Map(ctx, fn)` - Transform giá trị
- `Catch(ctx, fn)` - Xử lý lỗi
- `Finally(ctx, fn)` - Cleanup

**Đặc điểm:**
- ~183 dòng code
- Sử dụng generics (yêu cầu Go 1.21+, xem `go.mod`)
- Thread-safe: kết quả chỉ được ghi một lần (qua `resolveOnce`) rồi đóng `done` để "broadcast" cho mọi goroutine đang chờ - không dùng channel buffer-1 chỉ nhận được một lần như thiết kế cũ
- `ctx` truyền vào `Await`/`Then`/`Map`/`Catch`/`Finally` chỉ điều khiển việc **chờ**, không điều khiển việc **thực thi** của `fn`/`executor` (xem doc comment của `NewPromise`)

### 2. `pool.go` - Worker Pool
**Chứa:**
- `WorkerPool[T]` - Quản lý pool của workers (`taskQueue`, `done`, `closed atomic.Bool`, `closeOnce`)
- `task[T]` - Internal task struct (giữ `*Promise[T]`, không phải channel riêng)
- `NewWorkerPool(numWorkers)` - Tạo worker pool mới; `numWorkers <= 0` tự động fallback về 1
- `worker()` - Worker goroutine, ưu tiên rút cạn `taskQueue` trước khi thoát vì `done` đã đóng (tránh bỏ dở task còn trong queue)
- `executeTask()` - Thực thi task và xử lý panic (recover, trả `ErrTaskPanicked`)
- `Submit(fn)` - Gửi task vào queue, trả về Promise ngay. **Block** nếu queue đầy (backpressure) cho tới khi có chỗ hoặc pool đóng
- `Close()` - Đóng pool, chờ task đang chạy/còn trong queue chạy xong. An toàn khi gọi đồng thời với `Submit()`, gọi nhiều lần cũng an toàn (idempotent qua `closeOnce`)
- `Stats()` - Lấy thống kê

**Đặc điểm:**
- ~163 dòng code
- Panic recovery cho cả task lẫn (ở `types.go`) executor/fn của Promise thường
- `taskQueue` không bao giờ bị `close()` - chỉ `done` đóng đúng một lần - tránh panic "send on closed channel"
- `closed atomic.Bool` (không khóa) làm fast-path từ chối `Submit()` sau khi `Close()` return; xem giới hạn đã biết (khe hở race cực hẹp) ở comment của `Submit()` và README.md
- Configurable worker count

### 3. `combinators.go` - Promise Combinators
**Chứa:**
- `All()` - Chờ tất cả promises
- `Race()` - Chờ promise nhanh nhất
- `AllSettled()` - Chờ tất cả settle
- `Any()` - Chờ success đầu tiên
- `Sequence()` - Chạy tuần tự
- `Pool()` - Helper cho worker pool
- `PromiseStatus` & `Status` - Status types

**Đặc điểm:**
- ~225 dòng code
- Implementation đầy đủ JavaScript Promise API
- Thread-safe
- `All`/`Race`/`Any`/`AllSettled` không hủy các promise "thua cuộc" - goroutine chờ chúng vẫn chạy ngầm tới khi tự settle hoặc `ctx` hết hạn (xem doc comment của `All()` và mục Giới Hạn Đã Biết trong README.md)
- `Race()` với slice rỗng resolve ngay với zero-value, khác `Promise.race([])` của JS (treo mãi mãi)

### 4. `errors.go` - Error Handling
**Chứa:**
- `ErrTaskPanicked` - Panic error
- `ErrPoolClosed` - Pool closed error
- `ErrAllPromisesRejected` - All rejected error
- `AggregateError` - Container cho multiple errors
- `NewAggregateError()` - Tạo AggregateError
- Error methods: `Error()`, `Errors()` (trả về bản sao, không phải slice nội bộ), `Count()`

**Đặc điểm:**
- ~58 dòng code
- Formatted error messages
- Consistent error handling

### 5. `doc.go` - Documentation
**Chứa:**
- Package overview
- 7 ví dụ sử dụng chi tiết (Promise cơ bản, Worker Pool, Then/Catch, All, Race, NewPromiseWithExecutor, Pool helper)
- API reference
- Hiển thị qua `go doc github.com/phucps89/go-promise2`

**Đặc điểm:**
- Comprehensive documentation
- Examples cho phần lớn API chính
- Usage patterns

### 6. Test Files
**Chứa (4 file, tổng 41 test function + 2 fuzz target):**
- `promise_test.go` - Test chính cho Promise, WorkerPool, combinators (resolve/error/executor, chaining + ctx cancellation, Await lặp lại/đồng thời, panic recovery, race an toàn của Close/Submit, drain queue, Stats, combinator)
- `generic_types_test.go` - Test với generic type phức tạp (struct có con trỏ/slice/map)
- `backpressure_test.go` - Test `Submit()` block khi queue đầy và unblock đúng khi `Close()` được gọi
- `fuzz_test.go` - Fuzz test cho `WorkerPool` (round-trip kết quả) và `AggregateError` (format lỗi)

**Đặc điểm:**
- Toàn bộ pass với `go test -race`
- Khuyến khích chạy kèm `-race` vì package khai thác concurrency nhiều
- Clean test structure, tách theo nhóm quan tâm (concern) thay vì dồn hết vào 1 file

## 🎯 Design Patterns

### 1. Promise Pattern
```
Promise[T] -> Async Operation -> Result[T]
```
- Tạo promise (`NewPromise`) không block - công việc chạy nền trong goroutine riêng; `Await()` mới là điểm block để lấy kết quả
- Chainable operations (Then, Map, Catch)
- Error propagation

### 2. Worker Pool Pattern
```
TaskQueue -> Workers (goroutines) -> Result Channel
```
- Bounded parallelism
- Task queue buffer
- Graceful shutdown

### 3. Combinator Pattern
```
Promise[T]... -> Combinator -> Promise[Result]
```
- All: parallel execution with all success
- Race: first to complete
- AllSettled: all complete regardless
- Any: first success

## 📊 Complexity Analysis

### Promise Await
- Time: O(1)
- Space: O(1)
- Blocking operation

### WorkerPool Submit
- Time: O(1) khi queue còn chỗ trống
- Space: O(1)
- **Block** (không phải non-blocking) khi queue đã đầy, cho tới khi có chỗ trống hoặc pool đóng - đây là backpressure có chủ đích

### Combinators
- All: O(n) where n = number of promises
- Race: O(n) setup, O(1) resolution
- AllSettled: O(n)
- Any: O(n)

## 🔒 Thread Safety

- Promise: Safe. Kết quả chỉ ghi một lần qua `resolveOnce sync.Once`, sau đó đóng `done chan struct{}` để "broadcast" - mọi goroutine đang/sẽ `Await()` đều đọc được cùng giá trị cache, kể cả gọi đồng thời (race-tested)
- WorkerPool: Safe (`taskQueue` không bao giờ đóng, `closed atomic.Bool` + `select` trên `done`, `WaitGroup` cho graceful shutdown) - race-tested với `go test -race`, kể cả khi `Submit()`/`Close()` chạy đồng thời. Có 1 khe hở race cực hẹp đã biết và tài liệu hóa, xem README.md
- Combinators: Safe (`sync.Mutex`/`sync.Once` khi cần)

## 💡 Key Features

✅ **Generic Types** - Works with any type T
✅ **Context Support** - Cancellation cho việc chờ (`Await`), không cho việc thực thi `fn`
✅ **Panic Recovery** - Task/executor panic được recover, không crash process
✅ **Repeated Await** - `Await()`/`Then()`/... gọi nhiều lần hoặc đồng thời đều an toàn nhờ cache
✅ **Backpressure** - `Submit()` block khi queue đầy thay vì rớt task hoặc tạo goroutine không giới hạn
✅ **Race-tested** - Test suite chạy với `go test -race`, không phát hiện data race
✅ **Composable** - Chainable operations
✅ **Fuzz-tested** - `WorkerPool` và `AggregateError` có fuzz target riêng
✅ **Well Documented** - Examples, docs, và các giới hạn thiết kế đều được ghi rõ

## 🚀 Performance Characteristics

### Goroutine Count
- Promise: 1 goroutine cho mỗi promise được tạo qua `NewPromise`/`NewPromiseWithExecutor`
- WorkerPool: N goroutine cố định cho N worker (không tạo goroutine phụ mỗi lần `Submit()`)

### Latency
- Promise.Await: blocking, trả về gần như ngay khi promise đã resolve (đọc từ cache)
- WorkerPool.Submit: nhanh khi queue còn chỗ trống; **block** (backpressure) khi queue đầy cho tới khi có chỗ hoặc pool đóng

## 📈 Scalability

- **Small workloads** (1-10 concurrent): Use Promise
- **Medium workloads** (10-100 concurrent): Use WorkerPool with 4-16 workers
- **Large workloads** (100+ concurrent): Scale with multiple WorkerPools

## 🛠️ Maintenance

### Code Quality
- Clean, readable code
- Proper error handling
- Well-documented
- Comprehensive tests

### Extension Points
- Custom error types can extend AggregateError
- Custom worker pool behaviors via composition
- Custom combinators by using basic APIs

## 📝 Notes

- `Await()` block cho tới khi có kết quả hoặc `ctx` hết hạn; `Submit()` cũng block khi queue đầy (backpressure) - không phải mọi API đều non-blocking
- Context cancellation chỉ điều khiển việc chờ, không điều khiển việc thực thi bên trong `fn`/task
- Worker pool cần được `Close()` để giải phóng goroutine của các worker
- Promise operations là goroutine-safe, gọi lại nhiều lần được nhờ cache kết quả
- Xem mục "⚠️ Giới Hạn Đã Biết" trong README.md để biết các caveat thiết kế (goroutine leak khi dùng combinator với `ctx` không timeout, khe hở race cực hẹp giữa `Submit()`/`Close()`, `Race([])` khác JS, v.v.)
