package promise2

import (
	"sync"
	"sync/atomic"
)

// WorkerPool quản lý một pool của workers để xử lý tasks
type WorkerPool[T any] struct {
	taskQueue chan task[T]
	wg        sync.WaitGroup
	done      chan struct{}
	closeOnce sync.Once
	// closed là fast path: Submit() kiểm tra cờ này (không khóa, không chặn)
	// trước khi thử gửi task, nên bất kỳ Submit() nào xảy ra SAU KHI Close()
	// đã return đều bị từ chối ngay lập tức, chắc chắn - đúng kịch bản gây
	// treo Promise trước đây (Close() xong rồi Submit() vẫn lọt được task
	// vào queue dù không còn worker nào xử lý). Với Submit() đang chạy đồng
	// thời với Close(), độ an toàn thực sự nằm ở done trong select bên dưới,
	// chứ không phải cờ này.
	closed  atomic.Bool
	workers int
}

// task đại diện cho một công việc cần làm
type task[T any] struct {
	fn      func() (T, error)
	promise *Promise[T]
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

// worker là một worker routine xử lý tasks từ queue.
//
// Luôn thử nhận task từ taskQueue trước (nhánh default không chặn); chỉ khi
// queue rỗng mới chờ đồng thời trên taskQueue và done. Nhờ vậy, worker không
// bao giờ thoát do done đã đóng trong khi vẫn còn task nằm sẵn trong queue -
// tất cả task đã được Submit() chấp nhận đều được chạy trước khi pool đóng hẳn.
func (p *WorkerPool[T]) worker() {
	defer p.wg.Done()

	for {
		select {
		case t := <-p.taskQueue:
			p.executeTask(t)
			continue
		default:
		}

		select {
		case t := <-p.taskQueue:
			p.executeTask(t)
		case <-p.done:
			return
		}
	}
}

// executeTask thực thi một task và gửi kết quả
func (p *WorkerPool[T]) executeTask(t task[T]) {
	defer func() {
		if r := recover(); r != nil {
			t.promise.resolve(Result[T]{Err: ErrTaskPanicked})
		}
	}()

	val, err := t.fn()
	t.promise.resolve(Result[T]{Value: val, Err: err})
}

// Submit thêm một task vào queue và trả về Promise.
//
// taskQueue không bao giờ bị đóng (chỉ done mới bị đóng, đúng một lần, bởi
// Close()), nên việc gửi task vào đây luôn an toàn - không còn race
// "send on closed channel". Nếu queue đầy, Submit() block cho tới khi có chỗ
// trống hoặc pool đóng - đây là cơ chế backpressure bình thường của một
// worker pool có giới hạn, không cần goroutine riêng cho mỗi lần Submit và
// không giữ khóa nào trong lúc chờ, nên Close() không bao giờ bị một Submit()
// đang block làm treo theo (kể cả khi worker đang xử lý một task bị treo).
//
// Giới hạn đã biết (đánh đổi có chủ đích): giữa lúc closed.Load() thấy false
// và lúc select bên dưới thực sự chạy, Close() có thể chạy trọn vẹn (kể cả
// wg.Wait() return, mọi worker đã thoát). Khi đó select thấy cả hai case đều
// "ready" (taskQueue còn chỗ trống VÀ done đã đóng) nên có thể chọn ngẫu
// nhiên nhánh gửi vào taskQueue - task lọt vào queue nhưng không còn worker
// nào xử lý, khiến Promise tương ứng Await() treo vô thời hạn nếu ctx không
// có timeout. Xác suất cực thấp (chỉ xảy ra khi Submit() và Close() đua nhau
// đúng vài nano giây). Cách khắc phục triệt để (khóa toàn bộ đoạn kiểm tra +
// gửi bằng sync.RWMutex) đã được thử và bị loại bỏ vì gây deadlock nghiêm
// trọng hơn: Close() sẽ treo vĩnh viễn nếu một Submit() đang block chờ chỗ
// trống trong queue trong khi worker đang xử lý một task không bao giờ trả
// về. Khuyến nghị: luôn Await() bằng ctx có timeout, đặc biệt với các task
// được Submit() gần thời điểm pool có thể bị Close().
func (p *WorkerPool[T]) Submit(fn func() (T, error)) *Promise[T] {
	promise := newPromise[T]()

	if p.closed.Load() {
		promise.resolve(Result[T]{Err: ErrPoolClosed})
		return promise
	}

	t := task[T]{fn: fn, promise: promise}
	select {
	case p.taskQueue <- t:
	case <-p.done:
		promise.resolve(Result[T]{Err: ErrPoolClosed})
	}

	return promise
}

// Close đóng worker pool: ngừng nhận task mới và chờ tất cả task đang chạy
// hoặc còn nằm trong queue chạy xong. An toàn khi gọi đồng thời với Submit(),
// và có thể gọi nhiều lần (closeOnce đảm bảo done chỉ đóng một lần).
//
// closed được set trước khi đóng done, không cần khóa và không bao giờ chặn -
// nên Close() luôn đánh dấu pool là đã đóng ngay lập tức, kể cả khi có
// Submit() đang block chờ chỗ trống trong queue (Submit() đó sẽ được done
// "giải phóng" ngay, trả về ErrPoolClosed, thay vì chờ mãi task không bao
// giờ hoàn thành). wg.Wait() vẫn chờ các task đang thực sự chạy xong.
func (p *WorkerPool[T]) Close() error {
	p.closeOnce.Do(func() {
		p.closed.Store(true)
		close(p.done)
	})
	p.wg.Wait()
	return nil
}

// PoolStats chứa thống kê của worker pool
type PoolStats struct {
	NumWorkers    int
	QueueSize     int
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
