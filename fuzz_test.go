package promise2

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// FuzzWorkerPoolRoundTrip fuzz numWorkers và numTasks (kể cả giá trị âm/biên)
// để đảm bảo mọi task được Submit() đều round-trip đúng giá trị qua Await(),
// không panic, không treo - bất kể pool được cấu hình bất thường thế nào.
func FuzzWorkerPoolRoundTrip(f *testing.F) {
	f.Add(1, 1)
	f.Add(4, 50)
	f.Add(0, 10)
	f.Add(-3, 0)
	f.Add(1, 200)

	f.Fuzz(func(t *testing.T, numWorkers int, numTasks int) {
		if numTasks < 0 {
			numTasks = -numTasks
		}
		numTasks %= 200 // giới hạn để fuzz không tạo khối lượng việc khổng lồ
		if numWorkers > 32 || numWorkers < -32 {
			numWorkers %= 32
		}

		pool := NewWorkerPool[int](numWorkers)
		defer pool.Close()

		if pool.Stats().NumWorkers < 1 {
			t.Fatalf("NumWorkers phải luôn >= 1, got %d (input numWorkers=%d)", pool.Stats().NumWorkers, numWorkers)
		}

		promises := make([]*Promise[int], numTasks)
		for i := 0; i < numTasks; i++ {
			i := i
			promises[i] = pool.Submit(func() (int, error) { return i, nil })
		}

		for i, p := range promises {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			v, err := p.Await(ctx)
			cancel()
			if err != nil {
				t.Fatalf("task %d: unexpected error: %v", i, err)
			}
			if v != i {
				t.Fatalf("task %d: expected %d, got %d", i, i, v)
			}
		}
	})
}

// FuzzAggregateErrorFormatting fuzz message và số lượng lỗi để đảm bảo
// AggregateError.Error()/Count() không panic với bất kỳ nội dung/số lượng
// lỗi nào, kể cả chuỗi rỗng hoặc chứa ký tự đặc biệt.
func FuzzAggregateErrorFormatting(f *testing.F) {
	f.Add("boom", 3)
	f.Add("", 0)
	f.Add("lỗi tiếng Việt \n\t%s", 5)

	f.Fuzz(func(t *testing.T, msg string, count int) {
		if count < 0 {
			count = -count
		}
		count %= 50 // giới hạn để fuzz không tạo error list khổng lồ

		errs := make([]error, count)
		for i := range errs {
			errs[i] = fmt.Errorf("%s-%d", msg, i)
		}

		ae := NewAggregateError(errs)
		if ae.Count() != count {
			t.Fatalf("expected Count()=%d, got %d", count, ae.Count())
		}

		s := ae.Error() // không được panic
		if count == 0 && s != "aggregate error: no errors" {
			t.Fatalf("unexpected message for empty AggregateError: %q", s)
		}
	})
}
