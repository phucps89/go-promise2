package promise2

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrTaskPanicked xảy ra khi task panic trong worker
	ErrTaskPanicked = errors.New("task panicked during execution")

	// ErrPoolClosed xảy ra khi submit task vào pool đã đóng
	ErrPoolClosed = errors.New("worker pool is closed")

	// ErrAllPromisesRejected xảy ra khi dùng Any() và tất cả promises bị reject
	ErrAllPromisesRejected = errors.New("all promises were rejected")
)

// AggregateError chứa nhiều errors
type AggregateError struct {
	errors []error
}

// NewAggregateError tạo một AggregateError mới
func NewAggregateError(errs []error) *AggregateError {
	return &AggregateError{errors: errs}
}

// Error trả về string representation của AggregateError
func (ae *AggregateError) Error() string {
	if len(ae.errors) == 0 {
		return "aggregate error: no errors"
	}

	var sb strings.Builder
	sb.WriteString("aggregate error: ")
	for i, err := range ae.errors {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("[%d] %v", i, err))
	}
	return sb.String()
}

// Errors trả về bản sao của slice errors - sửa slice trả về không ảnh hưởng
// tới trạng thái nội bộ của AggregateError.
func (ae *AggregateError) Errors() []error {
	out := make([]error, len(ae.errors))
	copy(out, ae.errors)
	return out
}

// Count trả về số lượng errors
func (ae *AggregateError) Count() int {
	return len(ae.errors)
}
