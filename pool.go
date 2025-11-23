package promise2

import (
	"sync"
)

// WorkerPool quản lý một pool của workers để xử lý tasks
type WorkerPool[T any] struct {
	taskQueue chan task[T]
	wg        sync.WaitGroup
	done      chan struct{}
	workers   int
}

// task đại diện cho một công việc cần làm
type task[T any] struct {
	fn func() (T, error)
	ch chan Result[T]
}

// NewWorkerPool tạo một worker pool mới với số lượng workers
func NewWorkerPool[T any](numWorkers int) *WorkerPool[T] {
	if numWorkers <= 0 {
		numWorkers = 1
	}

	pool := &WorkerPool[T]{
		taskQueue: make(chan task[T], numWorkers*2),
		done:      make(chan struct{}),
		workers:   numWorkers,
	}

	// Khởi tạo workers
	for i := 0; i < numWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker là một worker routine xử lý tasks từ queue
func (p *WorkerPool[T]) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.done:
			return
		case t, ok := <-p.taskQueue:
			if !ok {
				return
			}
			p.executeTask(t)
		}
	}
}

// executeTask thực thi một task và gửi kết quả
func (p *WorkerPool[T]) executeTask(t task[T]) {
	defer func() {
		if r := recover(); r != nil {
			t.ch <- Result[T]{Err: ErrTaskPanicked}
		}
	}()

	val, err := t.fn()
	t.ch <- Result[T]{Value: val, Err: err}
}

// Submit thêm một task vào queue và trả về Promise
func (p *WorkerPool[T]) Submit(fn func() (T, error)) *Promise[T] {
	promise := &Promise[T]{
		resultChan: make(chan Result[T], 1),
	}

	go func() {
		t := task[T]{
			fn: fn,
			ch: promise.resultChan,
		}

		select {
		case p.taskQueue <- t:
			// Task đã được thêm vào queue
		case <-p.done:
			// Pool đã bị đóng
			promise.resultChan <- Result[T]{Err: ErrPoolClosed}
		}
	}()

	return promise
}

// Close đóng worker pool và chờ tất cả tasks hoàn thành
func (p *WorkerPool[T]) Close() error {
	close(p.done)
	close(p.taskQueue)
	p.wg.Wait()
	return nil
}

// PoolStats chứa thống kê của worker pool
type PoolStats struct {
	NumWorkers   int
	QueueSize    int
	QueueCapacity int
}

// Stats trả về thống kê hiện tại của pool
func (p *WorkerPool[T]) Stats() PoolStats {
	return PoolStats{
		NumWorkers:    p.workers,
		QueueSize:     len(p.taskQueue),
		QueueCapacity: cap(p.taskQueue),
	}
}
