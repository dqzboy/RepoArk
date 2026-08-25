package middleware

import (
	"net/http"
	"testing"
	"time"
)

func TestShouldLogAccess(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		route   string
		status  int
		latency time.Duration
		want    bool
	}{
		{name: "successful job polling is quiet", method: http.MethodGet, path: "/api/jobs/42", route: "/api/jobs/:id", status: http.StatusOK, latency: time.Millisecond, want: false},
		{name: "failed job polling is logged", method: http.MethodGet, path: "/api/jobs/42", route: "/api/jobs/:id", status: http.StatusUnauthorized, latency: time.Millisecond, want: true},
		{name: "slow job polling is logged", method: http.MethodGet, path: "/api/jobs/42", route: "/api/jobs/:id", status: http.StatusOK, latency: slowRequestThreshold, want: true},
		{name: "static asset is quiet", method: http.MethodGet, path: "/assets/app.js", status: http.StatusOK, latency: time.Millisecond, want: false},
		{name: "favicon is quiet", method: http.MethodGet, path: "/favicon.ico", status: http.StatusOK, latency: time.Millisecond, want: false},
		{name: "api read is logged", method: http.MethodGet, path: "/api/jobs", route: "/api/jobs", status: http.StatusOK, latency: time.Millisecond, want: true},
		{name: "write is logged", method: http.MethodPost, path: "/api/backup/run/gitcode", route: "/api/backup/run/:platform", status: http.StatusOK, latency: time.Millisecond, want: true},
		{name: "preflight is quiet", method: http.MethodOptions, path: "/api/jobs", status: http.StatusNoContent, latency: time.Millisecond, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldLogAccess(tt.method, tt.path, tt.route, tt.status, tt.latency); got != tt.want {
				t.Fatalf("shouldLogAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}
