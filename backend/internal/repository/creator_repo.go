package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/geo"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

const (
	nearbyMinResults = 10
	nearbyMaxRadius  = 100.0
)

type LocalCreatorRepo struct {
	db *sql.DB
}

func NewLocalCreatorRepo(db *sql.DB) *LocalCreatorRepo {
	return &LocalCreatorRepo{db: db}
}

// Upsert inserts or updates a creator profile. Deduplicates on (name, lat, lon).
func (r *LocalCreatorRepo) Upsert(req model.CreateCreatorRequest) (model.LocalCreator, error) {
	var linksJSON []byte
	if len(req.Links) > 0 && string(req.Links) != "null" {
		linksJSON = req.Links
	}

	var tagsJSON []byte
	if len(req.Tags) > 0 {
		var err error
		tagsJSON, err = json.Marshal(req.Tags)
		if err != nil {
			return model.LocalCreator{}, err
		}
	}

	row := r.db.QueryRow(`
		INSERT INTO local_creators (name, designation, bio, lat, lon, area_name, links, notable_works, tags, source, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (name, lat, lon) DO UPDATE SET
			designation   = EXCLUDED.designation,
			bio           = EXCLUDED.bio,
			area_name     = EXCLUDED.area_name,
			links         = EXCLUDED.links,
			notable_works = EXCLUDED.notable_works,
			tags          = EXCLUDED.tags,
			source        = EXCLUDED.source,
			image_url     = EXCLUDED.image_url
		RETURNING id, name, designation, bio, lat, lon, area_name, links, notable_works, tags, source, image_url, discovered_at, verified_at`,
		req.Name, req.Designation, nullString(req.Bio), req.Lat, req.Lon,
		nullString(req.AreaName), nullBytes(linksJSON), nullString(req.NotableWorks),
		nullBytes(tagsJSON), req.Source, nullString(req.ImageURL),
	)
	return scanCreator(row)
}

// ListNearby returns creators near lat/lon, expanding radius adaptively until ≥10 results.
// Returns the creators, the radius actually used, and any error.
func (r *LocalCreatorRepo) ListNearby(lat, lon, baseRadiusKm float64, limit int) ([]model.LocalCreator, float64, error) {
	tiers := []float64{baseRadiusKm, baseRadiusKm * 3, baseRadiusKm * 10}
	for _, radius := range tiers {
		if radius > nearbyMaxRadius {
			radius = nearbyMaxRadius
		}
		creators, err := r.queryWithRadius(lat, lon, radius, limit)
		if err != nil {
			return nil, 0, err
		}
		if len(creators) >= nearbyMinResults || radius >= nearbyMaxRadius {
			return creators, radius, nil
		}
	}
	return nil, 0, nil
}

func (r *LocalCreatorRepo) queryWithRadius(lat, lon, radiusKm float64, limit int) ([]model.LocalCreator, error) {
	minLat, maxLat, minLon, maxLon := geo.BoundingBox(lat, lon, radiusKm)
	rows, err := r.db.Query(`
		SELECT id, name, designation, bio, lat, lon, area_name, links, notable_works, tags, source, image_url, discovered_at, verified_at
		FROM local_creators
		WHERE lat BETWEEN $1 AND $2
		  AND lon BETWEEN $3 AND $4
		ORDER BY discovered_at DESC
		LIMIT $5`,
		minLat, maxLat, minLon, maxLon, limit*5,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.LocalCreator
	for rows.Next() {
		c, err := scanCreatorRow(rows)
		if err != nil {
			return nil, err
		}
		if c.Lat != nil && c.Lon != nil {
			if geo.HaversineKm(lat, lon, *c.Lat, *c.Lon) > radiusKm {
				continue
			}
		}
		results = append(results, c)
		if len(results) >= limit {
			break
		}
	}
	return results, rows.Err()
}

func scanCreator(row *sql.Row) (model.LocalCreator, error) {
	var c model.LocalCreator
	var bio, areaName, notableWorks, source, imageURL sql.NullString
	var linksJSON, tagsJSON []byte
	var verifiedAt sql.NullTime

	err := row.Scan(
		&c.ID, &c.Name, &c.Designation, &bio, &c.Lat, &c.Lon,
		&areaName, &linksJSON, &notableWorks, &tagsJSON,
		&source, &imageURL, &c.DiscoveredAt, &verifiedAt,
	)
	if err != nil {
		return model.LocalCreator{}, err
	}
	applyNullableCreatorFields(&c, bio, areaName, notableWorks, source, imageURL, linksJSON, tagsJSON, verifiedAt)
	return c, nil
}

func scanCreatorRow(rows *sql.Rows) (model.LocalCreator, error) {
	var c model.LocalCreator
	var bio, areaName, notableWorks, source, imageURL sql.NullString
	var linksJSON, tagsJSON []byte
	var verifiedAt sql.NullTime
	var discoveredAt time.Time

	err := rows.Scan(
		&c.ID, &c.Name, &c.Designation, &bio, &c.Lat, &c.Lon,
		&areaName, &linksJSON, &notableWorks, &tagsJSON,
		&source, &imageURL, &discoveredAt, &verifiedAt,
	)
	if err != nil {
		return model.LocalCreator{}, err
	}
	c.DiscoveredAt = discoveredAt
	applyNullableCreatorFields(&c, bio, areaName, notableWorks, source, imageURL, linksJSON, tagsJSON, verifiedAt)
	return c, nil
}

func applyNullableCreatorFields(c *model.LocalCreator, bio, areaName, notableWorks, source, imageURL sql.NullString, linksJSON, tagsJSON []byte, verifiedAt sql.NullTime) {
	c.Bio = bio.String
	c.AreaName = areaName.String
	c.NotableWorks = notableWorks.String
	c.Source = source.String
	c.ImageURL = imageURL.String
	if verifiedAt.Valid {
		t := verifiedAt.Time
		c.VerifiedAt = &t
	}
	if len(linksJSON) > 0 && string(linksJSON) != "null" {
		c.Links = json.RawMessage(linksJSON)
	}
	if len(tagsJSON) > 0 && string(tagsJSON) != "null" {
		_ = json.Unmarshal(tagsJSON, &c.Tags)
	}
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
