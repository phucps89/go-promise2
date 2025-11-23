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

// Promise là một wrapper cho async operation
type Promise[T any] struct {
	resultChan chan Result[T]
	once       sync.Once
}

// NewPromise tạo một Promise mới
func NewPromise[T any](fn func() (T, error)) *Promise[T] {
	p := &Promise[T]{
		resultChan: make(chan Result[T], 1),
	}

	go func() {
		val, err := fn()
		p.resultChan <- Result[T]{Value: val, Err: err}
	}()

	return p
}

// NewPromiseWithExecutor tạo một Promise với executor function
// Executor nhận resolve và reject callbacks
func NewPromiseWithExecutor[T any](
	executor func(resolve func(T), reject func(error)),
) *Promise[T] {
	p := &Promise[T]{
		resultChan: make(chan Result[T], 1),
	}

	go func() {
		resolve := func(val T) {
			p.once.Do(func() {
				p.resultChan <- Result[T]{Value: val, Err: nil}
			})
		}

		reject := func(err error) {
			p.once.Do(func() {
				p.resultChan <- Result[T]{Err: err}
			})
		}

		executor(resolve, reject)
	}()

	return p
}

// Await chờ kết quả của Promise
func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	select {
	case result := <-p.resultChan:
		return result.Value, result.Err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Then chuỗi Promise - thực thi fn khi Promise hiện tại hoàn thành
func (p *Promise[T]) Then(fn func(T) error) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		go func() {
			val, err := p.Await(context.Background())
			if err != nil {
				reject(err)
				return
			}

			if err := fn(val); err != nil {
				reject(err)
				return
			}

			resolve(val)
		}()
	})
}

// Map chuyển đổi giá trị của Promise
func (p *Promise[T]) Map(fn func(T) (T, error)) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		go func() {
			val, err := p.Await(context.Background())
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
		}()
	})
}

// Catch xử lý lỗi của Promise
func (p *Promise[T]) Catch(fn func(error) (T, error)) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		go func() {
			val, err := p.Await(context.Background())
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
		}()
	})
}

// Finally thực thi fn dù Promise thành công hay thất bại
func (p *Promise[T]) Finally(fn func()) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		go func() {
			val, err := p.Await(context.Background())
			fn()
			if err != nil {
				reject(err)
				return
			}
			resolve(val)
		}()
	})
}
