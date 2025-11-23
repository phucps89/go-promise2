package promise2

import (
	"context"
	"fmt"
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
	}).Then(func(val int) error {
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
	}).Map(func(val int) (int, error) {
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
	}).Catch(func(err error) (int, error) {
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
	}).Finally(func() {
		called = true
	})

	_, _ = promise.Await(context.Background())
	if !called {
		t.Fatal("Finally callback was not called")
	}
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
