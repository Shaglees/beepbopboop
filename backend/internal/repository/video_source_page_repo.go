package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// UpsertSourcePage stores or updates the raw crawl record for a discovered
// source page. source_url is the idempotency key across re-runs.
func (r *VideoRepo) UpsertSourcePage(p model.VideoSourcePage) error {
	_, err := r.db.Exec(`
		INSERT INTO video_source_pages (source_name, source_url, archive_url, raw_payload, last_error, fetched_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (source_url) DO UPDATE SET
			source_name = EXCLUDED.source_name,
			archive_url = EXCLUDED.archive_url,
			raw_payload = EXCLUDED.raw_payload,
			last_error  = EXCLUDED.last_error,
			fetched_at  = CURRENT_TIMESTAMP`,
		p.SourceName,
		p.SourceURL,
		nullString(p.ArchiveURL),
		nullRawJSON(p.RawPayload),
		nullString(p.LastError),
	)
	if err != nil {
		return fmt.Errorf("upsert video_source_page: %w", err)
	}
	return nil
}

func (r *VideoRepo) GetSourcePage(sourceURL string) (*model.VideoSourcePage, error) {
	var page model.VideoSourcePage
	var archiveURL, lastError sql.NullString
	var rawPayload []byte
	err := r.db.QueryRow(`
		SELECT source_name, source_url, archive_url, raw_payload, last_error, fetched_at
		FROM video_source_pages
		WHERE source_url = $1`, sourceURL).
		Scan(&page.SourceName, &page.SourceURL, &archiveURL, &rawPayload, &lastError, &page.FetchedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get video_source_page: %w", err)
	}
	page.ArchiveURL = archiveURL.String
	page.LastError = lastError.String
	if len(rawPayload) > 0 && string(rawPayload) != "null" {
		page.RawPayload = rawPayload
	}
	return &page, nil
}
