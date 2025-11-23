package promise2

import (
	"context"
	"sync"
)

// All chờ tất cả promises hoàn thành
// Nếu bất kỳ promise nào lỗi, trả về lỗi đó
func All[T any](ctx context.Context, promises ...*Promise[T]) *Promise[[]T] {
	return NewPromiseWithExecutor[[]T](func(resolve func([]T), reject func(error)) {
		n := len(promises)
		if n == 0 {
			resolve([]T{})
			return
		}

		results := make([]T, n)
		var mu sync.Mutex
		var errOnce sync.Once
		var wg sync.WaitGroup

		wg.Add(n)

		for i, promise := range promises {
			go func(idx int, p *Promise[T]) {
				defer wg.Done()

				val, err := p.Await(ctx)
				if err != nil {
					errOnce.Do(func() {
						reject(err)
					})
					return
				}

				mu.Lock()
				results[idx] = val
				mu.Unlock()
			}(i, promise)
		}

		go func() {
			wg.Wait()
			resolve(results)
		}()
	})
}

// Race trả về kết quả của promise hoàn thành đầu tiên
func Race[T any](ctx context.Context, promises ...*Promise[T]) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		if len(promises) == 0 {
			var zero T
			resolve(zero)
			return
		}

		var once sync.Once

		for _, promise := range promises {
			go func(p *Promise[T]) {
				val, err := p.Await(ctx)
				once.Do(func() {
					if err != nil {
						reject(err)
					} else {
						resolve(val)
					}
				})
			}(promise)
		}
	})
}

// AllSettled chờ tất cả promises settle (complete hoặc reject)
// Trả về slice của PromiseStatus cho từng promise
func AllSettled[T any](ctx context.Context, promises ...*Promise[T]) *Promise[[]PromiseStatus[T]] {
	return NewPromiseWithExecutor[[]PromiseStatus[T]](func(resolve func([]PromiseStatus[T]), reject func(error)) {
		n := len(promises)
		if n == 0 {
			resolve([]PromiseStatus[T]{})
			return
		}

		results := make([]PromiseStatus[T], n)
		var mu sync.Mutex
		var wg sync.WaitGroup

		wg.Add(n)

		for i, promise := range promises {
			go func(idx int, p *Promise[T]) {
				defer wg.Done()

				val, err := p.Await(ctx)

				mu.Lock()
				if err != nil {
					results[idx] = PromiseStatus[T]{
						Status: StatusRejected,
						Err:    err,
					}
				} else {
					results[idx] = PromiseStatus[T]{
						Status: StatusFulfilled,
						Value:  val,
					}
				}
				mu.Unlock()
			}(i, promise)
		}

		go func() {
			wg.Wait()
			resolve(results)
		}()
	})
}

// PromiseStatus chứa status và kết quả của một promise
type PromiseStatus[T any] struct {
	Status Status
	Value  T
	Err    error
}

// Status của promise
type Status string

const (
	StatusFulfilled Status = "fulfilled"
	StatusRejected  Status = "rejected"
)

// AnyResult chứa kết quả từ Any - giá trị thành công đầu tiên hoặc error
type AnyResult[T any] struct {
	Value T
	Err   error
}

// Any trả về kết quả của promise thành công đầu tiên
// Nếu tất cả promises reject, trả về AggregateError
func Any[T any](ctx context.Context, promises ...*Promise[T]) *Promise[T] {
	return NewPromiseWithExecutor[T](func(resolve func(T), reject func(error)) {
		n := len(promises)
		if n == 0 {
			reject(ErrAllPromisesRejected)
			return
		}

		var mu sync.Mutex
		errors := make([]error, 0, n)
		rejectedCount := 0
		var once sync.Once

		for _, promise := range promises {
			go func(p *Promise[T]) {
				val, err := p.Await(ctx)
				if err == nil {
					once.Do(func() {
						resolve(val)
					})
					return
				}

				mu.Lock()
				errors = append(errors, err)
				rejectedCount++
				allRejected := rejectedCount == n
				mu.Unlock()

				if allRejected {
					once.Do(func() {
						reject(NewAggregateError(errors))
					})
				}
			}(promise)
		}
	})
}

// Sequence thực thi promises theo thứ tự (từng cái một)
func Sequence[T any](ctx context.Context, promises ...*Promise[T]) *Promise[[]T] {
	return NewPromiseWithExecutor[[]T](func(resolve func([]T), reject func(error)) {
		n := len(promises)
		results := make([]T, n)

		for i, promise := range promises {
			val, err := promise.Await(ctx)
			if err != nil {
				reject(err)
				return
			}
			results[i] = val
		}

		resolve(results)
	})
}

// Pool chứa promises và chạy chúng với worker pool
func Pool[T any](ctx context.Context, pool *WorkerPool[T], tasks ...func() (T, error)) *Promise[[]T] {
	promises := make([]*Promise[T], len(tasks))
	for i, task := range tasks {
		promises[i] = pool.Submit(task)
	}
	return All(ctx, promises...)
}
