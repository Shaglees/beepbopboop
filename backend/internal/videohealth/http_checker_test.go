package videohealth_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/videohealth"
)

func TestHTTPChecker_CheckEmbed_MapsHTTPStatuses(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		want       string
	}{
		{name: "ok", statusCode: http.StatusOK, want: "ok"},
		{name: "unauthorized blocked", statusCode: http.StatusUnauthorized, want: "blocked"},
		{name: "forbidden blocked", statusCode: http.StatusForbidden, want: "blocked"},
		{name: "not found gone", statusCode: http.StatusNotFound, want: "gone"},
		{name: "gone gone", statusCode: http.StatusGone, want: "gone"},
		{name: "too many requests unknown", statusCode: http.StatusTooManyRequests, want: "unknown"},
		{name: "server error unknown", statusCode: http.StatusBadGateway, want: "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = fmt.Fprint(w, `{}`)
			}))
			defer srv.Close()

			checker := videohealth.NewHTTPChecker(srv.Client())
			checker.YouTubeOEmbed = srv.URL
			checker.VimeoOEmbed = srv.URL

			got, err := checker.CheckEmbed(context.Background(), model.Video{
				Provider: "youtube",
				WatchURL: "https://www.youtube.com/watch?v=test-id",
			})
			if err != nil {
				t.Fatalf("CheckEmbed: %v", err)
			}
			if got != tc.want {
				t.Fatalf("status mapping: got %q want %q", got, tc.want)
			}
		})
	}
}
