package repository

import (
	"container/list"
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// UserEmbeddingGetter loads user vectors. Implemented by *UserEmbeddingRepo and *EmbeddingCache.
type UserEmbeddingGetter interface {
	Get(ctx context.Context, userID string) (*model.UserEmbedding, error)
}

// EmbeddingCache adds LRU + TTL in front of a UserEmbeddingGetter (issue #44 TDD).
// Implemented in this package (not internal/ranking) to avoid an import cycle: post_repo
// already imports ranking for the two-tower Ranker.
type EmbeddingCache struct {
	mu         sync.Mutex
	loader     UserEmbeddingGetter
	maxEntries int
	ttl        time.Duration

	ll   *list.List
	idx  map[string]*list.Element
	hits int64
}

type cachedUserEmb struct {
	userID   string
	ue       *model.UserEmbedding
	deadline time.Time
}

// NewEmbeddingCache wraps the DB with a UserEmbeddingRepo loader (issue #44 signature).
func NewEmbeddingCache(db *sql.DB, maxEntries int, ttl time.Duration) *EmbeddingCache {
	return NewEmbeddingCacheFromLoader(NewUserEmbeddingRepo(db), maxEntries, ttl)
}

// NewEmbeddingCacheFromLoader wraps an existing getter (e.g. shared repo instance).
func NewEmbeddingCacheFromLoader(loader UserEmbeddingGetter, maxEntries int, ttl time.Duration) *EmbeddingCache {
	if maxEntries < 1 {
		maxEntries = 1
	}
	if ttl < time.Millisecond {
		ttl = time.Minute
	}
	return &EmbeddingCache{
		loader:     loader,
		maxEntries: maxEntries,
		ttl:        ttl,
		ll:         list.New(),
		idx:        make(map[string]*list.Element),
	}
}

// DBHits returns how many times the loader was invoked (cache misses and expired entries).
func (c *EmbeddingCache) DBHits() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return int(c.hits)
}

// Get returns a copy of the user embedding or (nil, nil) if none exists.
func (c *EmbeddingCache) Get(ctx context.Context, userID string) (*model.UserEmbedding, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.idx[userID]; ok {
		cu := el.Value.(*cachedUserEmb)
		if time.Now().Before(cu.deadline) {
			c.ll.MoveToFront(el)
			return cloneUserEmb(cu.ue), nil
		}
		c.ll.Remove(el)
		delete(c.idx, userID)
	}

	c.hits++
	ue, err := c.loader.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if ue == nil {
		return nil, nil
	}
	stored := cloneUserEmb(ue)

	for c.ll.Len() >= c.maxEntries {
		back := c.ll.Back()
		if back == nil {
			break
		}
		old := back.Value.(*cachedUserEmb)
		delete(c.idx, old.userID)
		c.ll.Remove(back)
	}

	cu := &cachedUserEmb{userID: userID, ue: stored, deadline: time.Now().Add(c.ttl)}
	front := c.ll.PushFront(cu)
	c.idx[userID] = front
	return cloneUserEmb(stored), nil
}

func cloneUserEmb(ue *model.UserEmbedding) *model.UserEmbedding {
	if ue == nil {
		return nil
	}
	out := *ue
	if len(ue.Embedding) > 0 {
		out.Embedding = make([]float32, len(ue.Embedding))
		copy(out.Embedding, ue.Embedding)
	}
	return &out
}
