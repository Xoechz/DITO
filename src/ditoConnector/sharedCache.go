package dito

import (
	"hash/fnv"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type entityInfoCacheItem struct {
	traceId         pcommon.TraceID
	rootSpanId      pcommon.SpanID
	jobSpanIds      map[pcommon.SpanID]pcommon.SpanID //ingested job SpanID -> new SpanID
	samplingCounter int
	createdAt       time.Time
}

type jobCacheItem struct {
	jobSpan    *ptrace.Span
	receivedAt time.Time
}

type entityInfoCacheShard struct {
	entityInfoCache map[string]*entityInfoCacheItem
	mu              sync.RWMutex
}
type jobCacheShard struct {
	jobCache map[pcommon.SpanID]*jobCacheItem // ingested job SpanID -> ingested job span
	mu       sync.RWMutex
}

type sharedCache struct {
	jobAddQueue      chan *jobCacheItem
	entityShards     []*entityInfoCacheShard
	jobShards        []*jobCacheShard
	entityShardCount int
	jobShardCount    int
	maxAge           time.Duration
}

func newSharedCache(entityShardCount int, jobShardCount int, maxAge time.Duration) *sharedCache {
	entityShards := make([]*entityInfoCacheShard, entityShardCount)
	jobShards := make([]*jobCacheShard, jobShardCount)

	for i := 0; i < entityShardCount; i++ {
		entityShards[i] = &entityInfoCacheShard{
			entityInfoCache: make(map[string]*entityInfoCacheItem),
		}
	}

	for i := 0; i < jobShardCount; i++ {
		jobShards[i] = &jobCacheShard{
			jobCache: make(map[pcommon.SpanID]*jobCacheItem),
		}
	}

	return &sharedCache{
		jobAddQueue:      make(chan *jobCacheItem, 1000),
		entityShards:     entityShards,
		jobShards:        jobShards,
		entityShardCount: entityShardCount,
		jobShardCount:    jobShardCount,
		maxAge:           maxAge,
	}
}

func (sc *sharedCache) shardForEntity(entityKey string) *entityInfoCacheShard {
	h := fnv.New32a()
	h.Write([]byte(entityKey))
	hash := h.Sum32()

	return sc.entityShards[hash%uint32(sc.entityShardCount)]
}

func (sc *sharedCache) shardForJob(jobSpanID pcommon.SpanID) *jobCacheShard {
	h := fnv.New32a()
	h.Write(jobSpanID[:])
	hash := h.Sum32()

	return sc.jobShards[hash%uint32(sc.jobShardCount)]
}

func (sc *sharedCache) getOrCreateEntityEntry(entityKey string) (*entityInfoCacheItem, bool) {
	shard := sc.shardForEntity(entityKey)

	shard.mu.RLock()
	entry, exists := shard.entityInfoCache[entityKey]
	shard.mu.RUnlock()

	if exists {
		return entry, false
	}

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// re-check in case it was created while acquiring the lock
	entry, exists = shard.entityInfoCache[entityKey]

	if exists {
		return entry, false
	}

	newEntry := &entityInfoCacheItem{
		traceId:         generateTraceID(),
		rootSpanId:      generateSpanID(),
		jobSpanIds:      make(map[pcommon.SpanID]pcommon.SpanID),
		samplingCounter: 0,
		createdAt:       time.Now(),
	}

	shard.entityInfoCache[entityKey] = newEntry
	return newEntry, true
}

func (sc *sharedCache) getJobSpan(jobSpanID pcommon.SpanID) (*ptrace.Span, bool) {
	shard := sc.shardForJob(jobSpanID)

	shard.mu.RLock()
	jobItem, exists := shard.jobCache[jobSpanID]
	shard.mu.RUnlock()

	if exists {
		return jobItem.jobSpan, true
	}

	addItem := <-sc.jobAddQueue
	if addItem == nil {
		return nil, false
	}

	for addItem != nil {
		shard := sc.shardForJob(addItem.jobSpan.SpanID())
		shard.mu.Lock()
		shard.jobCache[addItem.jobSpan.SpanID()] = addItem
		shard.mu.Unlock()

		if addItem.jobSpan.SpanID() == jobSpanID {
			jobItem = addItem
		}

		addItem = <-sc.jobAddQueue
	}

	return jobItem.jobSpan, jobItem != nil
}

func (sc *sharedCache) addJobSpan(jobSpan *ptrace.Span, receivedAt time.Time) {
	sc.jobAddQueue <- &jobCacheItem{
		jobSpan:    jobSpan,
		receivedAt: receivedAt,
	}
}
