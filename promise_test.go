package promise2

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewPromise kiểm tra tạo promise cơ bản
func TestNewPromise(t *testing.T) {
	promise := NewPromise(func() (int, error) {
		return 42, nil
	})

	result, err := promise.Await(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
}

// TestNewPromiseWithError kiểm tra promise trả về lỗi
func TestNewPromiseWithError(t *testing.T) {
	expectedErr := fmt.Errorf("test error")
	promise := NewPromise(func() (int, error) {
		return 0, expectedErr
	})

	_, err := promise.Await(context.Background())
	if err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

// TestNewPromiseWithExecutor kiểm tra executor pattern
func TestNewPromiseWithExecutor(t *testing.T) {
	promise := NewPromiseWithExecutor[string](func(resolve func(string), reject func(error)) {
		resolve("success")
	})

	result, err := promise.Await(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Fatalf("expected 'success', got '%s'", result)
	}
}

// TestPromiseThen kiểm tra Then chaining
func TestPromiseThen(t *testing.T) {
	promise := NewPromise(func() (int, error) {
		return 10, nil
	}).Then(context.Background(), func(val int) error {
		if val != 10 {
			return fmt.Errorf("expected 10, got %d", val)
		}
		return nil
	})

	_, err := promise.Await(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestPromiseMap kiểm tra Map transformation
func TestPromiseMap(t *testing.T) {
	promise := NewPromise(func() (int, error) {
		return 5, nil
	}).Map(context.Background(), func(val int) (int, error) {
		return val * 2, nil
	})

	result, err := promise.Await(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 10 {
		t.Fatalf("expected 10, got %d", result)
	}
}

// TestPromiseCatch kiểm tra error handling
func TestPromiseCatch(t *testing.T) {
	promise := NewPromise(func() (int, error) {
		return 0, fmt.Errorf("original error")
	}).Catch(context.Background(), func(err error) (int, error) {
		return 99, nil // Recover with default value
	})

	result, err := promise.Await(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 99 {
		t.Fatalf("expected 99, got %d", result)
	}
}

// TestPromiseFinally kiểm tra Finally
func TestPromiseFinally(t *testing.T) {
	called := false
	promise := NewPromise(func() (int, error) {
		return 42, nil
	}).Finally(context.Background(), func() {
		called = true
	})

	_, _ = promise.Await(context.Background())
	if !called {
		t.Fatal("Finally callback was not called")
	}
}

// TestPromiseAwaitMultipleTimes kiểm tra Await() có thể gọi nhiều lần và
// luôn trả về cùng kết quả đã cache (trước đây sẽ treo ở lần gọi thứ 2).
func TestPromiseAwaitMultipleTimes(t *testing.T) {
	promise := NewPromise(func() (int, error) {
		return 42, nil
	})

	for i := 0; i < 3; i++ {
		result, err := promise.Await(context.Background())
		if err != nil {
			t.Fatalf("await #%d: unexpected error: %v", i, err)
		}
		if result != 42 {
			t.Fatalf("await #%d: expected 42, got %d", i, result)
		}
	}
}

// TestPromiseAwaitConcurrentWithThen kiểm tra Await() trực tiếp và Then()
// trên cùng một Promise có thể chạy đồng thời mà không tranh giành kết quả
// (trước đây bên thua cuộc race sẽ treo vô thời hạn).
func TestPromiseAwaitConcurrentWithThen(t *testing.T) {
	promise := NewPromise(func() (int, error) {
		return 7, nil
	})

	var wg sync.WaitGroup
	errs := make([]error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, errs[0] = promise.Await(context.Background())
	}()
	go func() {
		defer wg.Done()
		thenPromise := promise.Then(context.Background(), func(v int) error { return nil })
		_, errs[1] = thenPromise.Await(context.Background())
	}()
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: unexpected error: %v", i, err)
		}
	}
}

// TestPromisePanicRecovered kiểm tra panic trong fn của NewPromise được
// recover và trả về ErrTaskPanicked thay vì làm crash cả process.
func TestPromisePanicRecovered(t *testing.T) {
	promise := NewPromise(func() (int, error) {
		panic("boom")
	})

	_, err := promise.Await(context.Background())
	if !errors.Is(err, ErrTaskPanicked) {
		t.Fatalf("expected ErrTaskPanicked, got %v", err)
	}
}

// TestPromiseWithExecutorPanicRecovered kiểm tra panic trong executor của
// NewPromiseWithExecutor cũng được recover.
func TestPromiseWithExecutorPanicRecovered(t *testing.T) {
	promise := NewPromiseWithExecutor[int](func(resolve func(int), reject func(error)) {
		panic("boom")
	})

	_, err := promise.Await(context.Background())
	if !errors.Is(err, ErrTaskPanicked) {
		t.Fatalf("expected ErrTaskPanicked, got %v", err)
	}
}

// TestPromiseThenRespectsContext kiểm tra Then() dừng chờ khi ctx bị hủy
// thay vì treo vô thời hạn nếu Promise gốc không bao giờ resolve.
func TestPromiseThenRespectsContext(t *testing.T) {
	promise := NewPromiseWithExecutor[int](func(resolve func(int), reject func(error)) {
		// không bao giờ resolve/reject
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	chained := promise.Then(ctx, func(v int) error { return nil })

	_, err := chained.Await(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

// TestPromiseMapRespectsContext kiểm tra Map() cũng dừng chờ khi ctx bị hủy,
// giống Then() - cùng cơ chế nhưng chưa được test riêng.
func TestPromiseMapRespectsContext(t *testing.T) {
	promise := NewPromiseWithExecutor[int](func(resolve func(int), reject func(error)) {
		// không bao giờ resolve/reject
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	chained := promise.Map(ctx, func(v int) (int, error) { return v, nil })

	_, err := chained.Await(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

// TestPromiseCatchRespectsContext kiểm tra Catch() không treo vô thời hạn
// khi ctx bị hủy - fn nhận đúng context.DeadlineExceeded như một lỗi bình
// thường (Catch bắt mọi lỗi, kể cả lỗi do ctx hết hạn khi chờ promise gốc).
func TestPromiseCatchRespectsContext(t *testing.T) {
	promise := NewPromiseWithExecutor[int](func(resolve func(int), reject func(error)) {
		// không bao giờ resolve/reject
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var gotErr error
	chained := promise.Catch(ctx, func(err error) (int, error) {
		gotErr = err
		return 0, err // không "nuốt" lỗi, trả lại nguyên trạng
	})

	_, err := chained.Await(context.Background())
	if !errors.Is(gotErr, context.DeadlineExceeded) {
		t.Fatalf("expected Catch's fn to receive context.DeadlineExceeded, got %v", gotErr)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

// TestPromiseFinallyRespectsContext kiểm tra Finally() không treo vô thời
// hạn khi ctx bị hủy. fn cleanup vẫn phải được gọi - ctx timeout cũng là một
// dạng "thất bại" của promise gốc, và Finally cam kết chạy dù thành công hay
// thất bại - rồi lỗi timeout vẫn được truyền tiếp qua Await().
func TestPromiseFinallyRespectsContext(t *testing.T) {
	promise := NewPromiseWithExecutor[int](func(resolve func(int), reject func(error)) {
		// không bao giờ resolve/reject
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	called := false
	chained := promise.Finally(ctx, func() { called = true })

	_, err := chained.Await(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if !called {
		t.Fatal("Finally callback should still run on ctx timeout - Finally always runs regardless of outcome")
	}
}

// TestPromiseAwaitManyGoroutines dội hàng loạt goroutine cùng Await() một
// promise để tăng độ tin cậy cho tính đúng đắn của resolveOnce/done dưới tải
// cao, không chỉ 2 goroutine như TestPromiseAwaitConcurrentWithThen.
func TestPromiseAwaitManyGoroutines(t *testing.T) {
	promise := NewPromise(func() (int, error) {
		return 123, nil
	})

	const n = 200
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			v, err := promise.Await(context.Background())
			if err != nil || v != 123 {
				t.Errorf("expected (123, nil), got (%d, %v)", v, err)
			}
		}()
	}
	wg.Wait()
}

// TestWorkerPoolBasic kiểm tra worker pool cơ bản
func TestWorkerPoolBasic(t *testing.T) {
	pool := NewWorkerPool[int](2)
	defer pool.Close()

	p1 := pool.Submit(func() (int, error) {
		return 1, nil
	})

	p2 := pool.Submit(func() (int, error) {
		return 2, nil
	})

	r1, _ := p1.Await(context.Background())
	r2, _ := p2.Await(context.Background())

	if r1 != 1 || r2 != 2 {
		t.Fatalf("expected [1, 2], got [%d, %d]", r1, r2)
	}
}

// TestWorkerPoolClosed kiểm tra submit vào pool đã đóng
func TestWorkerPoolClosed(t *testing.T) {
	pool := NewWorkerPool[int](1)
	pool.Close()

	p := pool.Submit(func() (int, error) {
		return 1, nil
	})

	_, err := p.Await(context.Background())
	if err != ErrPoolClosed {
		t.Fatalf("expected ErrPoolClosed, got %v", err)
	}
}

// TestWorkerPoolCloseRaceWithSubmit dội Submit() và Close() đồng thời để
// đảm bảo không còn panic "send on closed channel" (chạy với -race để bắt
// data race). Trước đây taskQueue bị đóng trong khi Submit() vẫn có thể
// đang gửi vào đó.
func TestWorkerPoolCloseRaceWithSubmit(t *testing.T) {
	for attempt := 0; attempt < 50; attempt++ {
		pool := NewWorkerPool[int](4)
		var wg sync.WaitGroup

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				pool.Submit(func() (int, error) { return n, nil })
			}(i)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.Close()
		}()

		wg.Wait()
	}
}

// TestWorkerPoolDrainsQueueOnClose kiểm tra các task đã được Submit() chấp
// nhận vào queue đều được chạy xong dù Close() được gọi ngay sau đó, thay vì
// bị bỏ dở khi worker chọn nhánh done trong lúc queue vẫn còn task.
func TestWorkerPoolDrainsQueueOnClose(t *testing.T) {
	pool := NewWorkerPool[int](1)

	const n = 8 // > queue capacity (numWorkers*2 = 2), sẽ có task chờ trong queue
	promises := make([]*Promise[int], n)
	for i := 0; i < n; i++ {
		i := i
		promises[i] = pool.Submit(func() (int, error) { return i, nil })
	}

	go pool.Close()

	for i, p := range promises {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		val, err := p.Await(ctx)
		cancel()
		if err != nil {
			t.Fatalf("task %d: expected no error (task should run before pool closes), got %v", i, err)
		}
		if val != i {
			t.Fatalf("task %d: expected %d, got %d", i, i, val)
		}
	}
}

// TestWorkerPoolTaskPanicRecovered kiểm tra panic bên trong task chạy qua
// WorkerPool được executeTask recover và trả về ErrTaskPanicked, thay vì làm
// crash worker goroutine (và không làm treo các task khác trong pool).
func TestWorkerPoolTaskPanicRecovered(t *testing.T) {
	pool := NewWorkerPool[int](2)
	defer pool.Close()

	panicking := pool.Submit(func() (int, error) {
		panic("boom")
	})
	ok := pool.Submit(func() (int, error) {
		return 5, nil
	})

	_, err := panicking.Await(context.Background())
	if !errors.Is(err, ErrTaskPanicked) {
		t.Fatalf("expected ErrTaskPanicked, got %v", err)
	}

	v, err := ok.Await(context.Background())
	if err != nil || v != 5 {
		t.Fatalf("pool should keep working after a panicked task: got (%d, %v)", v, err)
	}
}

// TestWorkerPoolCloseWaitsForRunningTask kiểm tra Close() thực sự chờ task
// đang chạy dở (không chỉ task còn nằm trong queue) hoàn thành trước khi
// trả về - wg.Wait() phải block cho tới khi executeTask() thực sự xong.
func TestWorkerPoolCloseWaitsForRunningTask(t *testing.T) {
	pool := NewWorkerPool[int](1)

	started := make(chan struct{})
	var finished atomic.Bool
	p := pool.Submit(func() (int, error) {
		close(started)
		time.Sleep(150 * time.Millisecond)
		finished.Store(true)
		return 1, nil
	})

	<-started
	pool.Close() // phải block cho tới khi task trên chạy xong

	if !finished.Load() {
		t.Fatal("Close() returned before the running task finished")
	}

	v, err := p.Await(context.Background())
	if err != nil || v != 1 {
		t.Fatalf("expected (1, nil), got (%d, %v)", v, err)
	}
}

// TestWorkerPoolCloseIdempotent kiểm tra Close() có thể gọi nhiều lần, kể cả
// đồng thời từ nhiều goroutine, mà không panic (closeOnce phải chặn được
// việc đóng done hai lần).
func TestWorkerPoolCloseIdempotent(t *testing.T) {
	pool := NewWorkerPool[int](2)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := pool.Close(); err != nil {
				t.Errorf("unexpected error from Close(): %v", err)
			}
		}()
	}
	wg.Wait()
}

// TestWorkerPoolStats kiểm tra Stats() phản ánh đúng số worker và trạng thái
// queue.
func TestWorkerPoolStats(t *testing.T) {
	pool := NewWorkerPool[int](3)
	defer pool.Close()

	stats := pool.Stats()
	if stats.NumWorkers != 3 {
		t.Fatalf("expected NumWorkers=3, got %d", stats.NumWorkers)
	}
	if stats.QueueCapacity != 6 { // numWorkers * 2
		t.Fatalf("expected QueueCapacity=6, got %d", stats.QueueCapacity)
	}

	block := make(chan struct{})
	// Chiếm hết cả 3 worker để các task tiếp theo phải nằm trong queue.
	for i := 0; i < 3; i++ {
		pool.Submit(func() (int, error) { <-block; return 0, nil })
	}
	for i := 0; i < 2; i++ {
		pool.Submit(func() (int, error) { return 0, nil })
	}

	// Cho worker/queue kịp ổn định trạng thái trước khi đọc Stats().
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && pool.Stats().QueueSize != 2 {
		time.Sleep(time.Millisecond)
	}

	if size := pool.Stats().QueueSize; size != 2 {
		t.Fatalf("expected QueueSize=2, got %d", size)
	}

	close(block)
}

// TestNewWorkerPoolInvalidWorkerCount kiểm tra numWorkers <= 0 được fallback
// về 1 worker thay vì tạo pool không hoạt động.
func TestNewWorkerPoolInvalidWorkerCount(t *testing.T) {
	for _, n := range []int{0, -1, -100} {
		pool := NewWorkerPool[int](n)
		if pool.Stats().NumWorkers != 1 {
			t.Fatalf("numWorkers=%d: expected fallback to 1, got %d", n, pool.Stats().NumWorkers)
		}

		p := pool.Submit(func() (int, error) { return 42, nil })
		v, err := p.Await(context.Background())
		pool.Close()
		if err != nil || v != 42 {
			t.Fatalf("numWorkers=%d: expected (42, nil), got (%d, %v)", n, v, err)
		}
	}
}

// TestAllPromisesReusableAfterCombinator kiểm tra các promise nguồn vẫn
// Await() được bình thường (trực tiếp và nhiều lần) sau khi đã được All()
// tiêu thụ - trước đây All() sẽ "ăn" mất giá trị của promise nguồn.
func TestAllPromisesReusableAfterCombinator(t *testing.T) {
	p1 := NewPromise(func() (int, error) { return 1, nil })
	p2 := NewPromise(func() (int, error) { return 2, nil })

	if _, err := All(context.Background(), p1, p2).Await(context.Background()); err != nil {
		t.Fatalf("unexpected error from All: %v", err)
	}

	// Sau khi All() đã Await() cả p1 và p2, gọi trực tiếp trên từng promise
	// (nhiều lần) vẫn phải trả về đúng giá trị thay vì treo.
	for i := 0; i < 2; i++ {
		v1, err := p1.Await(context.Background())
		if err != nil || v1 != 1 {
			t.Fatalf("p1 await #%d: expected (1, nil), got (%d, %v)", i, v1, err)
		}
		v2, err := p2.Await(context.Background())
		if err != nil || v2 != 2 {
			t.Fatalf("p2 await #%d: expected (2, nil), got (%d, %v)", i, v2, err)
		}
	}
}

// TestAll kiểm tra All combinator
func TestAll(t *testing.T) {
	p1 := NewPromise(func() (int, error) {
		return 1, nil
	})
	p2 := NewPromise(func() (int, error) {
		return 2, nil
	})
	p3 := NewPromise(func() (int, error) {
		return 3, nil
	})

	results, err := All(context.Background(), p1, p2, p3).Await(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []int{1, 2, 3}
	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(results))
	}

	for i, r := range results {
		if r != expected[i] {
			t.Fatalf("expected %d at index %d, got %d", expected[i], i, r)
		}
	}
}

// TestAllWithError kiểm tra All khi có error
func TestAllWithError(t *testing.T) {
	p1 := NewPromise(func() (int, error) {
		return 1, nil
	})
	p2 := NewPromise(func() (int, error) {
		return 0, fmt.Errorf("test error")
	})

	_, err := All(context.Background(), p1, p2).Await(context.Background())
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

// TestRace kiểm tra Race combinator
func TestRace(t *testing.T) {
	p1 := NewPromise(func() (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 1, nil
	})
	p2 := NewPromise(func() (int, error) {
		return 2, nil
	})

	result, _ := Race(context.Background(), p1, p2).Await(context.Background())
	if result != 2 {
		t.Fatalf("expected 2 (faster), got %d", result)
	}
}

// TestAllSettled kiểm tra AllSettled combinator
func TestAllSettled(t *testing.T) {
	p1 := NewPromise(func() (string, error) {
		return "success", nil
	})
	p2 := NewPromise(func() (string, error) {
		return "", fmt.Errorf("error")
	})

	results, _ := AllSettled(context.Background(), p1, p2).Await(context.Background())

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Status != StatusFulfilled {
		t.Fatal("expected first promise to be fulfilled")
	}

	if results[1].Status != StatusRejected {
		t.Fatal("expected second promise to be rejected")
	}
}

// TestAny kiểm tra Any combinator
func TestAny(t *testing.T) {
	p1 := NewPromise(func() (int, error) {
		return 0, fmt.Errorf("error 1")
	})
	p2 := NewPromise(func() (int, error) {
		return 42, nil
	})
	p3 := NewPromise(func() (int, error) {
		return 0, fmt.Errorf("error 3")
	})

	result, _ := Any(context.Background(), p1, p2, p3).Await(context.Background())
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
}

// TestAnyAllRejected kiểm tra Any khi tất cả reject
func TestAnyAllRejected(t *testing.T) {
	p1 := NewPromise(func() (int, error) {
		return 0, fmt.Errorf("error 1")
	})
	p2 := NewPromise(func() (int, error) {
		return 0, fmt.Errorf("error 2")
	})

	_, err := Any(context.Background(), p1, p2).Await(context.Background())
	if err == nil {
		t.Fatal("expected AggregateError")
	}

	if ae, ok := err.(*AggregateError); ok {
		if ae.Count() != 2 {
			t.Fatalf("expected 2 errors, got %d", ae.Count())
		}
	} else {
		t.Fatal("expected AggregateError type")
	}
}

// TestSequence kiểm tra Sequence combinator
func TestSequence(t *testing.T) {
	p1 := NewPromise(func() (int, error) {
		return 1, nil
	})
	p2 := NewPromise(func() (int, error) {
		return 2, nil
	})
	p3 := NewPromise(func() (int, error) {
		return 3, nil
	})

	results, _ := Sequence(context.Background(), p1, p2, p3).Await(context.Background())

	expected := []int{1, 2, 3}
	for i, r := range results {
		if r != expected[i] {
			t.Fatalf("expected %d at index %d, got %d", expected[i], i, r)
		}
	}
}

// TestPool kiểm tra Pool helper
func TestPool(t *testing.T) {
	pool := NewWorkerPool[int](2)
	defer pool.Close()

	tasks := []func() (int, error){
		func() (int, error) { return 1, nil },
		func() (int, error) { return 2, nil },
		func() (int, error) { return 3, nil },
	}

	results, _ := Pool(context.Background(), pool, tasks...).Await(context.Background())

	expected := []int{1, 2, 3}
	for i, r := range results {
		if r != expected[i] {
			t.Fatalf("expected %d at index %d, got %d", expected[i], i, r)
		}
	}
}

// TestContextCancellation kiểm tra context cancellation
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	promise := NewPromise(func() (int, error) {
		time.Sleep(1 * time.Second)
		return 42, nil
	})

	_, err := promise.Await(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
