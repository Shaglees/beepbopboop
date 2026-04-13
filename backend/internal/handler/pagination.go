package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// parsePagination extracts cursor and limit from query params.
func parsePagination(r *http.Request) (cursor string, limit int, err error) {
	cursor = r.URL.Query().Get("cursor")
	if cursor != "" {
		// Validate cursor format: "2006-01-02T15:04:05Z|42"
		parts := strings.SplitN(cursor, "|", 2)
		if len(parts) != 2 {
			return "", 0, fmt.Errorf("invalid_cursor")
		}
		if _, err := time.Parse(time.RFC3339, parts[0]); err != nil {
			return "", 0, fmt.Errorf("invalid_cursor")
		}
		if _, err := strconv.ParseInt(parts[1], 10, 64); err != nil {
			return "", 0, fmt.Errorf("invalid_cursor")
		}
	}

	limit = defaultLimit
	if ls := r.URL.Query().Get("limit"); ls != "" {
		l, err := strconv.Atoi(ls)
		if err == nil && l > 0 {
			limit = l
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return cursor, limit, nil
}
