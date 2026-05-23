package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Entry struct {
	Data      json.RawMessage `json:"data"`
	FetchedAt time.Time       `json:"fetched_at"`
}

type Cache struct {
	entries map[string]Entry
	path    string
}

func New(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	c := &Cache{
		entries: make(map[string]Entry),
		path:    filepath.Join(dir, "cache.json"),
	}
	_ = c.load()
	return c, nil
}

func (c *Cache) Get(key string, ttl time.Duration, dest any) bool {
	e, ok := c.entries[key]
	if !ok {
		return false
	}
	if time.Since(e.FetchedAt) > ttl {
		return false
	}
	return json.Unmarshal(e.Data, dest) == nil
}

// GetStale returns cached data even if expired, for offline fallback
func (c *Cache) GetStale(key string, dest any) (bool, time.Time) {
	e, ok := c.entries[key]
	if !ok {
		return false, time.Time{}
	}
	return json.Unmarshal(e.Data, dest) == nil, e.FetchedAt
}

func (c *Cache) Set(key string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.entries[key] = Entry{Data: b, FetchedAt: time.Now()}
	return c.save()
}

func (c *Cache) load() error {
	f, err := os.Open(c.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&c.entries)
}

func (c *Cache) save() error {
	f, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(c.entries)
}
