package promise2

// Hướng dẫn sử dụng package promise2
//
// Package promise2 cung cấp một cách đơn giản và dễ hiểu để làm việc với goroutines
// tương tự như JavaScript Promises. Nó hỗ trợ worker pool để quản lý concurrent tasks hiệu quả.
//
// Ví dụ 1: Tạo một Promise cơ bản
//
//	promise := promise2.NewPromise(func() (string, error) {
//		// Làm việc bất đồng bộ
//		return "Hello", nil
//	})
//
//	result, err := promise.Await(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(result) // "Hello"
//
// Ví dụ 2: Sử dụng Worker Pool
//
//	pool := promise2.NewWorkerPool[int](4) // 4 workers
//	defer pool.Close()
//
//	// Submit tasks vào pool
//	p1 := pool.Submit(func() (int, error) {
//		time.Sleep(1 * time.Second)
//		return 42, nil
//	})
//
//	p2 := pool.Submit(func() (int, error) {
//		return 100, nil
//	})
//
//	results, err := promise2.All(context.Background(), p1, p2).Await(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(results) // [42, 100]
//
// Ví dụ 3: Promise.then() và .catch()
//
//	promise := promise2.NewPromise(func() (int, error) {
//		return 10, nil
//	}).Then(func(val int) error {
//		fmt.Println("Got:", val)
//		return nil
//	}).Catch(func(err error) (int, error) {
//		fmt.Println("Error:", err)
//		return 0, nil
//	})
//
//	_, _ = promise.Await(context.Background())
//
// Ví dụ 4: Promise.All() - chờ tất cả promises
//
//	tasks := []func() (string, error){
//		func() (string, error) { return "task1", nil },
//		func() (string, error) { return "task2", nil },
//		func() (string, error) { return "task3", nil },
//	}
//
//	promises := make([]*promise2.Promise[string], len(tasks))
//	for i, task := range tasks {
//		promises[i] = promise2.NewPromise(task)
//	}
//
//	results, err := promise2.All(context.Background(), promises...).Await(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(results) // [task1, task2, task3]
//
// Ví dụ 5: Promise.Race() - kết quả của promise hoàn thành đầu tiên
//
//	p1 := promise2.NewPromise(func() (int, error) {
//		time.Sleep(1 * time.Second)
//		return 1, nil
//	})
//
//	p2 := promise2.NewPromise(func() (int, error) {
//		return 2, nil
//	})
//
//	winner, _ := promise2.Race(context.Background(), p1, p2).Await(context.Background())
//	fmt.Println(winner) // 2 (p2 faster)
//
// Ví dụ 6: NewPromiseWithExecutor - kiểm soát resolve/reject
//
//	promise := promise2.NewPromiseWithExecutor[string](func(resolve func(string), reject func(error)) {
//		// Làm việc bất đồng bộ
//		result, err := someAsyncOperation()
//		if err != nil {
//			reject(err)
//			return
//		}
//		resolve(result)
//	})
//
//	result, err := promise.Await(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(result)
//
// Ví dụ 7: Worker Pool với Pool helper
//
//	pool := promise2.NewWorkerPool[int](4)
//	defer pool.Close()
//
//	tasks := []func() (int, error){
//		func() (int, error) { return 1, nil },
//		func() (int, error) { return 2, nil },
//		func() (int, error) { return 3, nil },
//	}
//
//	results, _ := promise2.Pool(context.Background(), pool, tasks...).Await(context.Background())
//	fmt.Println(results) // [1, 2, 3]
//
// Các API chính:
//
// Promise:
//   - NewPromise(fn) - Tạo promise từ function
//   - NewPromiseWithExecutor(executor) - Tạo promise với executor
//   - Await(ctx) - Chờ kết quả (blocking)
//   - Then(fn) - Chuỗi promise
//   - Map(fn) - Transform giá trị
//   - Catch(fn) - Xử lý lỗi
//   - Finally(fn) - Cleanup
//
// WorkerPool:
//   - NewWorkerPool[T](numWorkers) - Tạo worker pool
//   - Submit(fn) - Gửi task vào pool
//   - Close() - Đóng pool
//   - Stats() - Lấy thống kê
//
// Combinators:
//   - All(...promises) - Chờ tất cả
//   - Race(...promises) - Chờ cái nhanh nhất
//   - AllSettled(...promises) - Chờ tất cả settle
//   - Any(...promises) - Chờ cái thành công đầu tiên
//   - Sequence(...promises) - Chạy tuần tự
//   - Pool(ctx, pool, ...tasks) - Chạy tasks trong pool
