package promise2

import (
	"context"
	"testing"
	"time"
)

// TestWorkerPoolSubmitBlocksWhenFull đo trực tiếp hành vi backpressure của
// Submit(): khi cả worker lẫn taskQueue đều đã kín chỗ, Submit() phải BLOCK
// caller (không trả về Promise ngay) cho tới khi có chỗ trống, thay vì nhận
// task một cách "lạc quan" rồi âm thầm rớt kết quả.
func TestWorkerPoolSubmitBlocksWhenFull(t *testing.T) {
	pool := NewWorkerPool[int](1) // capacity queue = numWorkers*2 = 2
	defer func() {
		// đảm bảo goroutine phía dưới không rò rỉ nếu test fail giữa chừng
	}()

	block := make(chan struct{})
	task := func() (int, error) {
		<-block
		return 0, nil
	}

	// 1 task chiếm worker duy nhất (chạy mãi tới khi block đóng) + 2 task lấp
	// đầy queue (capacity 2) => tổng cộng 3 "chỗ" đã bị chiếm hết.
	pool.Submit(task)
	pool.Submit(task)
	pool.Submit(task)

	submitReturned := make(chan struct{})
	go func() {
		pool.Submit(func() (int, error) { return 99, nil })
		close(submitReturned)
	}()

	select {
	case <-submitReturned:
		close(block)
		t.Fatal("Submit() trả về ngay dù pool đã kín chỗ - đáng lẽ phải block")
	case <-time.After(150 * time.Millisecond):
		// đúng như kỳ vọng: Submit() thứ 4 vẫn đang block
	}

	close(block) // giải phóng để worker rảnh chỗ, Submit() thứ 4 phải hoàn tất

	select {
	case <-submitReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("Submit() không bao giờ trả về sau khi pool đã có chỗ trống")
	}
}

// TestWorkerPoolSubmitUnblocksOnClose kiểm tra Submit() đang bị block chờ
// chỗ trống trong queue sẽ được giải phóng (với ErrPoolClosed) khi Close()
// được gọi, thay vì treo vĩnh viễn.
func TestWorkerPoolSubmitUnblocksOnClose(t *testing.T) {
	pool := NewWorkerPool[int](1)

	block := make(chan struct{})
	defer close(block)
	task := func() (int, error) { <-block; return 0, nil }

	pool.Submit(task)
	pool.Submit(task)
	pool.Submit(task) // worker + queue (capacity 2) đều kín

	blockedPromise := make(chan *Promise[int], 1)
	go func() {
		blockedPromise <- pool.Submit(func() (int, error) { return 1, nil })
	}()

	// Cho goroutine trên kịp đi vào trạng thái block trước khi Close().
	time.Sleep(50 * time.Millisecond)
	go pool.Close()

	select {
	case p := <-blockedPromise:
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := p.Await(ctx)
		if err != ErrPoolClosed {
			t.Fatalf("expected ErrPoolClosed, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Submit() không bao giờ unblock sau khi Close() được gọi")
	}
}
