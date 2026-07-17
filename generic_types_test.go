package promise2

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

var errBoom = errors.New("boom")

// payload là một struct phức tạp (có slice, map) dùng để kiểm tra Promise và
// WorkerPool hoạt động đúng với type parameter không phải primitive.
type payload struct {
	ID    int
	Name  string
	Tags  []string
	Meta  map[string]int
	Child *payload
}

func samplePayload() payload {
	return payload{
		ID:   7,
		Name: "order-7",
		Tags: []string{"urgent", "retail"},
		Meta: map[string]int{"retries": 2},
		Child: &payload{
			ID:   8,
			Name: "order-8",
		},
	}
}

// TestPromiseWithStructType kiểm tra Promise[T] hoạt động đúng khi T là một
// struct phức tạp (có con trỏ, slice, map) - đặc biệt là Await() nhiều lần
// vẫn trả về giá trị cache giống hệt nhau (so sánh sâu bằng reflect.DeepEqual).
func TestPromiseWithStructType(t *testing.T) {
	want := samplePayload()

	promise := NewPromise(func() (payload, error) {
		return want, nil
	})

	for i := 0; i < 3; i++ {
		got, err := promise.Await(context.Background())
		if err != nil {
			t.Fatalf("await #%d: unexpected error: %v", i, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("await #%d: got %+v, want %+v", i, got, want)
		}
	}
}

// TestPromiseChainWithStructType kiểm tra Then/Map/Catch hoạt động đúng khi
// transform giữa các struct value.
func TestPromiseChainWithStructType(t *testing.T) {
	base := samplePayload()

	promise := NewPromise(func() (payload, error) {
		return base, nil
	}).Map(context.Background(), func(p payload) (payload, error) {
		p.Tags = append(append([]string{}, p.Tags...), "processed")
		return p, nil
	})

	got, err := promise.Await(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantTags := []string{"urgent", "retail", "processed"}
	if !reflect.DeepEqual(got.Tags, wantTags) {
		t.Fatalf("got tags %v, want %v", got.Tags, wantTags)
	}

	// Promise gốc (base) không được thay đổi bởi Map (map trả về giá trị mới).
	if len(base.Tags) != 2 {
		t.Fatalf("Map() must not mutate the original payload's Tags, got %v", base.Tags)
	}
}

// TestWorkerPoolWithStructType kiểm tra WorkerPool[T] Submit/Await đúng với
// struct type, kể cả khi zero-value (task lỗi) cũng phải là zero-value đúng
// kiểu của T (payload{}), không panic hay lẫn dữ liệu giữa các task.
func TestWorkerPoolWithStructType(t *testing.T) {
	pool := NewWorkerPool[payload](2)
	defer pool.Close()

	want := samplePayload()
	ok := pool.Submit(func() (payload, error) { return want, nil })
	failed := pool.Submit(func() (payload, error) { return payload{}, errBoom })

	got, err := ok.Await(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	zero, err := failed.Await(context.Background())
	if err != errBoom {
		t.Fatalf("expected errBoom, got %v", err)
	}
	if !reflect.DeepEqual(zero, payload{}) {
		t.Fatalf("expected zero-value payload on error, got %+v", zero)
	}
}
