package middleware

import (
	"context"
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// RequestLogRecorder is the subset of store.Store that Logging needs to
// persist a per-request row  narrowed to an interface so main.go can pass
// *db.Postgres directly without middleware importing the store package.
type RequestLogRecorder interface {
	RecordRequestLog(ctx context.Context, userID *int64, method, path string, instanceName *string, status int, durationMs float64) error
}

type requestLogCapture struct {
	userID *int64
}

type requestLogCaptureKeyType struct{}

var requestLogCaptureKey = requestLogCaptureKeyType{}

// setRequestLogUserID is called by RequireAuth once it has authenticated a
// request, so the eventual request-log row records who made it.
func setRequestLogUserID(ctx context.Context, userID int64) {
	if capture, ok := ctx.Value(requestLogCaptureKey).(*requestLogCapture); ok {
		capture.userID = &userID
	}
}

func Logging(recorder RequestLogRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()

			capture := &requestLogCapture{}
			r = r.WithContext(context.WithValue(r.Context(), requestLogCaptureKey, capture))

			next.ServeHTTP(rec, r)

			duration := time.Since(start)
			log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, duration)

			if recorder == nil {
				return
			}
			var instanceName *string
			if id := r.PathValue("id"); id != "" {
				instanceName = &id
			}
			durationMs := float64(duration) / float64(time.Millisecond)
			if err := recorder.RecordRequestLog(r.Context(), capture.userID, r.Method, r.URL.Path, instanceName, rec.status, durationMs); err != nil {
				log.Printf("request log write failed: method=%s path=%s err=%v", r.Method, r.URL.Path, err)
			}
		})
	}
}
