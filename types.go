package promise2

import (
	"context"
	"sync"
)

// Result chứa kết quả hoặc lỗi của một task
type Result[T any] struct {
	Value T
	Err   error
}

// Promise là một wrapper cho async operation.
//
// Kết quả được cache lại sau khi resolve lần đầu tiên (bằng resolveOnce),
// nên Await/Then/Map/Catch/Finally có thể được gọi nhiều lần - kể cả đồng thời
// từ nhiều goroutine - và luôn nhận được cùng một giá trị, giống hành vi
// .then() có thể gọi lặp lại của Promise trong JavaScript.
type Promise[T any] struct {
	done        chan struct{}
	result      Result[T]
	resolveOnce sync.Once
}

func newPromise[T any]() *Promise[T] {
	return &Promise[T]{done: make(chan struct{})}
}

// resolve gắn kết quả cho Promise. Chỉ lần gọi đầu tiên có tác dụng - các lần
// gọi sau (ví dụ executor gọi resolve rồi reject, hoặc panic sau khi đã resolve)
// bị bỏ qua, đúng theo ngữ nghĩa "settle một lần" của Promise.
func (p *Promise[T]) resolve(r Result[T]) {
	p.resolveOnce.Do(func() {
		p.result = r
		close(p.done)
	})
}

// NewPromise tạo một Promise mới.
//
// Lưu ý kiến trúc: fn không nhận context.Context, nên không có cách nào để
// báo cho fn biết là nên dừng sớm - ctx truyền vào Await()/Then()/... chỉ
// điều khiển việc CHỜ, không điều khiển việc THỰC THI của fn. Nếu cần fn tự
// kiểm tra hủy (vd để dừng sớm khi All()/Race() đã có kết quả), tự viết fn
// với closure bắt ctx riêng, ví dụ:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	p := promise2.NewPromise(func() (int, error) {
//		select {
//		case <-ctx.Done():
//			return 0, ctx.Err()
//		case <-time.After(time.Second):
//			return 42, nil
//		}
//	})
func NewPromise[T any](fn func() (T, error)) *Promise[T] {
	p := newPromise[T]()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.resolve(Result[T]{Err: ErrTaskPanicked})
			}
		}()

		val, err := fn()
		p.resolve(Result[T]{Value: val, Err: err})
	}()

	return p
}

// NewPromiseWithExecutor tạo một Promise với executor function
// Executor nhận resolve và reject callbacks
func NewPromiseWithExecutor[T any](
	executor func(resolve func(T), reject func(error)),
) *Promise[T] {
	p := newPromise[T]()

	resolve := func(val T) {
		p.resolve(Result[T]{Value: val})
	}
	reject := func(err error) {
		p.resolve(Result[T]{Err: err})
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.resolve(Result[T]{Err: ErrTaskPanicked})
			}
		}()

		executor(resolve, reject)
	}()

	return p
}

// Await chờ kết quả của Promise. An toàn khi gọi nhiều lần (kể cả đồng thời
// từ nhiều goroutine) - lần gọi đầu tiên chờ kết quả thật, các lần sau đọc
// thẳng giá trị đã cache.
func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	select {
	case <-p.done:
		return p.result.Value, p.result.Err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Then chuỗi Promise - thực thi fn khi Promise hiện tại hoàn thành.
// ctx được truyền xuống Await bên trong, nên hủy ctx sẽ dừng chờ thay vì
// treo vô thời hạn nếu Promise gốc không bao giờ resolve.
func (p *Promise[T]) Then(ctx context.Context, fn func(T) error) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		val, err := p.Await(ctx)
		if err != nil {
			reject(err)
			return
		}

		if err := fn(val); err != nil {
			reject(err)
			return
		}

		resolve(val)
	})
}

// Map chuyển đổi giá trị của Promise
func (p *Promise[T]) Map(ctx context.Context, fn func(T) (T, error)) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		val, err := p.Await(ctx)
		if err != nil {
			reject(err)
			return
		}

		newVal, err := fn(val)
		if err != nil {
			reject(err)
			return
		}

		resolve(newVal)
	})
}

// Catch xử lý lỗi của Promise
func (p *Promise[T]) Catch(ctx context.Context, fn func(error) (T, error)) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		val, err := p.Await(ctx)
		if err == nil {
			resolve(val)
			return
		}

		newVal, err := fn(err)
		if err != nil {
			reject(err)
			return
		}

		resolve(newVal)
	})
}

// Finally thực thi fn dù Promise thành công hay thất bại
func (p *Promise[T]) Finally(ctx context.Context, fn func()) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		val, err := p.Await(ctx)
		fn()
		if err != nil {
			reject(err)
			return
		}
		resolve(val)
	})
}
