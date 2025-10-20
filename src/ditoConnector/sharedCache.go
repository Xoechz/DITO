package dito

import (
	"hash/fnv"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type JobState int

const (
	JobStateNotFound JobState = iota
	JobStateFound
	JobStateCreated
)

type spanWithResource struct {
	span     *ptrace.Span
	resource *pcommon.Resource
}

type entityWorkItem struct {
	entityKey  string
	sr         *spanWithResource
	receivedAt time.Time
}

type entityInfo struct {
	traceId         pcommon.TraceID
	rootSpanId      pcommon.SpanID
	samplingCounter int
}

type entityInfoCacheItem struct {
	traceId         pcommon.TraceID
	rootSpanId      pcommon.SpanID
	samplingCounter int
	createdAt       time.Time
}

type jobCacheItem struct {
	rs            *spanWithResource
	newJobSpanIds map[string]pcommon.SpanID // entityKey -> new SpanID
	receivedAt    time.Time
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
	logger       *zap.Logger
	config       Config
	jobAddQueue  chan *jobCacheItem
	entityShards []*entityInfoCacheShard
	jobShards    []*jobCacheShard
	shardCount   uint32
	messageQueue chan *entityWorkItem
}

func newSharedCache(cfg *Config, logger *zap.Logger) *sharedCache {
	entityShards := make([]*entityInfoCacheShard, cfg.CacheShardCount)
	jobShards := make([]*jobCacheShard, cfg.CacheShardCount)

	for i := 0; i < cfg.CacheShardCount; i++ {
		entityShards[i] = &entityInfoCacheShard{
			entityInfoCache: make(map[string]*entityInfoCacheItem),
		}
		jobShards[i] = &jobCacheShard{
			jobCache: make(map[pcommon.SpanID]*jobCacheItem),
		}
	}

	return &sharedCache{
		logger:       logger,
		config:       *cfg,
		jobAddQueue:  make(chan *jobCacheItem, 1000),
		entityShards: entityShards,
		jobShards:    jobShards,
		shardCount:   uint32(cfg.CacheShardCount),
		messageQueue: make(chan *entityWorkItem, cfg.QueueSize),
	}
}

func (sc *sharedCache) hashIndex(data []byte) uint32 {
	h := fnv.New32a()
	h.Write(data)
	return h.Sum32() % sc.shardCount
}

func (sc *sharedCache) shardForEntity(entityKey string) *entityInfoCacheShard {
	return sc.entityShards[sc.hashIndex([]byte(entityKey))]
}

func (sc *sharedCache) shardForJob(jobSpanID pcommon.SpanID) *jobCacheShard {
	return sc.jobShards[sc.hashIndex(jobSpanID[:])]
}

func (sc *sharedCache) getOrCreateEntityEntry(entityKey string) (entityInfo, bool) {
	shard := sc.shardForEntity(entityKey)

	shard.mu.RLock()
	entry, exists := shard.entityInfoCache[entityKey]
	shard.mu.RUnlock()

	if exists {
		entry.samplingCounter++

		return entityInfo{
			traceId:         entry.traceId,
			rootSpanId:      entry.rootSpanId,
			samplingCounter: entry.samplingCounter,
		}, false
	}

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// re-check in case it was created while acquiring the lock
	entry, exists = shard.entityInfoCache[entityKey]

	if exists {
		entry.samplingCounter++

		return entityInfo{
			traceId:         entry.traceId,
			rootSpanId:      entry.rootSpanId,
			samplingCounter: entry.samplingCounter,
		}, false
	}

	newEntry := &entityInfoCacheItem{
		traceId:         generateTraceID(),
		rootSpanId:      generateSpanID(),
		samplingCounter: 0,
		createdAt:       time.Now(),
	}

	shard.entityInfoCache[entityKey] = newEntry

	return entityInfo{
		traceId:         newEntry.traceId,
		rootSpanId:      newEntry.rootSpanId,
		samplingCounter: newEntry.samplingCounter,
	}, true
}

func (sc *sharedCache) drainJobQueue() {
	// Drain pending job spans without blocking.
	for {
		select {
		case addItem := <-sc.jobAddQueue:
			if addItem != nil {
				shardAdd := sc.shardForJob(addItem.rs.span.SpanID())
				shardAdd.mu.Lock()
				shardAdd.jobCache[addItem.rs.span.SpanID()] = addItem
				shardAdd.mu.Unlock()
			}
		default:
			return
		}
	}
}

func (sc *sharedCache) getJobSpan(entitySpan *ptrace.Span, entityKey string) (*spanWithResource, pcommon.SpanID, JobState) {
	sc.drainJobQueue()
	jobSpanID := entitySpan.ParentSpanID()

	shard := sc.shardForJob(jobSpanID)
	shard.mu.RLock()
	_, exists := shard.jobCache[jobSpanID]
	shard.mu.RUnlock()

	if !exists {
		baggageJobSpanId, exists := entitySpan.Attributes().Get(sc.config.BaggageJobKey)
		if !exists {
			return nil, pcommon.SpanID{}, JobStateNotFound
		}

		var err error
		jobSpanID, err = getSpanIDFromHexString(baggageJobSpanId.AsString())

		if err != nil {
			sc.logger.Error("Failed to parse job span ID from baggage", zap.String("baggageValue", baggageJobSpanId.AsString()), zap.Error(err))
			return nil, pcommon.SpanID{}, JobStateNotFound
		}
	}

	shard = sc.shardForJob(jobSpanID)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	jobItem, exists := shard.jobCache[jobSpanID]

	if !exists {
		return nil, pcommon.SpanID{}, JobStateNotFound
	}

	// return a copy of the job span
	returnSpan := jobItem.rs.copy()

	// If the job span was already used for this entityKey, return the same spanID
	newSpanId, exists := jobItem.newJobSpanIds[entityKey]
	if exists {
		return returnSpan, newSpanId, JobStateFound
	}

	newSpanId = generateSpanID()
	jobItem.newJobSpanIds[entityKey] = newSpanId

	return returnSpan, newSpanId, JobStateCreated
}

func (sc *sharedCache) addJobSpan(jobSpan *ptrace.Span, resource *pcommon.Resource, receivedAt time.Time) {
	select {
	case sc.jobAddQueue <- &jobCacheItem{
		rs:            &spanWithResource{span: jobSpan, resource: resource},
		receivedAt:    receivedAt,
		newJobSpanIds: make(map[string]pcommon.SpanID),
	}:
	default:
		sc.logger.Error("Job span queue full, dropping job span", zap.String("spanID", jobSpan.SpanID().String()))
	}
}

func (sc *sharedCache) ingestTraces(td ptrace.Traces, cfg *Config) error {
	rss := td.ResourceSpans()
	now := time.Now()

	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		resource := rs.Resource()
		ilss := rs.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			for k := 0; k < ils.Spans().Len(); k++ {
				span := ils.Spans().At(k)

				entityKey, isEntity := span.Attributes().Get(cfg.EntityKey)
				if isEntity {
					sc.messageQueue <- &entityWorkItem{
						entityKey:  entityKey.AsString(),
						sr:         &spanWithResource{span: &span, resource: &resource},
						receivedAt: now,
					}
				}

				_, isJob := span.Attributes().Get(cfg.JobKey)
				if isJob {
					sc.addJobSpan(&span, &resource, now)
				}
			}
		}
	}

	return nil
}

func (sc *sharedCache) sweep() {
	now := time.Now()

	for _, sh := range sc.entityShards {
		sh.mu.Lock()
		for k, e := range sh.entityInfoCache {
			if e.createdAt.Add(sc.config.EntityCacheDuration).Before(now) {
				delete(sh.entityInfoCache, k)
			}
		}
		sh.mu.Unlock()
	}

	for _, sh := range sc.jobShards {
		sh.mu.Lock()
		for k, e := range sh.jobCache {
			if e.receivedAt.Add(sc.config.MaxCacheDuration).Before(now) {
				delete(sh.jobCache, k)
			}
		}
		sh.mu.Unlock()
	}
}

func (sr *spanWithResource) copy() *spanWithResource {
	newSpan := ptrace.NewSpan()
	sr.span.CopyTo(newSpan)
	newResource := pcommon.NewResource()
	sr.resource.CopyTo(newResource)

	return &spanWithResource{
		span:     &newSpan,
		resource: &newResource,
	}
}
