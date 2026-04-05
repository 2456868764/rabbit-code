package webfetchtool

import (
	"sync"
	"time"
)

// Mirrors utils.ts URL_CACHE: 15m TTL, ~50MB byte budget.
const (
	urlCacheTTL    = 15 * time.Minute
	urlCacheMaxB   = 50 * 1024 * 1024
	urlCacheMinRec = 1
)

type urlCachePayload struct {
	bytes         int
	code          int
	codeText      string
	content       string
	contentType   string
	persistedPath string
	persistedSize int
}

type urlCacheEntry struct {
	expires time.Time
	size    int
	payload urlCachePayload
}

var (
	urlCacheMu sync.Mutex
	urlCache   = make(map[string]*urlCacheEntry)
	urlBytes   int
)

func urlCacheGet(key string) (urlCachePayload, bool) {
	urlCacheMu.Lock()
	defer urlCacheMu.Unlock()
	e, ok := urlCache[key]
	if !ok || time.Now().After(e.expires) {
		if ok {
			urlBytes -= e.size
			delete(urlCache, key)
		}
		return urlCachePayload{}, false
	}
	return e.payload, true
}

func urlCacheSet(key string, p urlCachePayload) {
	contentSize := len(p.content)
	if contentSize < urlCacheMinRec {
		contentSize = urlCacheMinRec
	}
	entrySize := contentSize

	urlCacheMu.Lock()
	defer urlCacheMu.Unlock()
	if old, ok := urlCache[key]; ok {
		urlBytes -= old.size
		delete(urlCache, key)
	}
	for urlBytes+entrySize > urlCacheMaxB && len(urlCache) > 0 {
		urlCacheEvictOldest()
	}
	urlCache[key] = &urlCacheEntry{
		expires: time.Now().Add(urlCacheTTL),
		size:    entrySize,
		payload: p,
	}
	urlBytes += entrySize
}

func urlCacheEvictOldest() {
	var oldestKey string
	var oldest time.Time
	for k, e := range urlCache {
		if oldestKey == "" || e.expires.Before(oldest) {
			oldest = e.expires
			oldestKey = k
		}
	}
	if oldestKey == "" {
		return
	}
	e := urlCache[oldestKey]
	urlBytes -= e.size
	delete(urlCache, oldestKey)
}

// ClearURLCacheForTest clears the fetch URL cache.
func ClearURLCacheForTest() {
	urlCacheMu.Lock()
	urlCache = make(map[string]*urlCacheEntry)
	urlBytes = 0
	urlCacheMu.Unlock()
}
