package tuner

import (
	"sort"
	"sync"
	"time"
)

var (
	DefaultReplicaMemoryMaxSize   = 2000
	DefaultReplicaMemoryRetention = time.Hour
)

type ReplicaSnapshot struct {
	Timestamp time.Time
	Replicas  int32
}

type ReplicaMemory interface {
	Add(event ReplicaSnapshot)
	GetMemorySince(since time.Time) []ReplicaSnapshot
	GetDeltaSince(since time.Time) int32
	GetFirstSnapshotAfter(since time.Time) *ReplicaSnapshot
}

type replicaMemory struct {
	events    []ReplicaSnapshot
	maxSize   int
	retention time.Duration
	mu        sync.RWMutex
}

func NewSimpleReplicaMemory(maxSize int, retention time.Duration) *replicaMemory {
	return &replicaMemory{
		events:    make([]ReplicaSnapshot, 0, maxSize), // Pre-allocate capacity to maxSize
		maxSize:   maxSize,
		retention: retention,
	}
}

func (s *replicaMemory) Add(event ReplicaSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.retention)

	// Filter expired events in-place to avoid allocating new slice
	n := 0
	for _, e := range s.events {
		if !e.Timestamp.Before(cutoff) {
			s.events[n] = e
			n++
		}
	}
	s.events = s.events[:n]

	// Add new event
	if len(s.events) == s.maxSize {
		// Shift elements left when at capacity
		copy(s.events, s.events[1:])
		s.events[s.maxSize-1] = event
	} else {
		s.events = append(s.events, event)
	}
	// Sort events by timestamp
	sort.Slice(s.events, func(i, j int) bool {
		return s.events[i].Timestamp.Before(s.events[j].Timestamp)
	})
}

func (s *replicaMemory) GetMemorySince(since time.Time) []ReplicaSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]ReplicaSnapshot, 0, len(s.events))
	for _, e := range s.events {
		if !e.Timestamp.Before(since) {
			events = append(events, e)
		}
	}
	return events
}

func (s *replicaMemory) GetDeltaSince(since time.Time) int32 {
	events := s.GetMemorySince(since)
	delta := int32(0)
	if len(events) < 2 {
		return 0
	}

	lastEvent := events[len(events)-1]
	firstEvent := events[0]

	delta = lastEvent.Replicas - firstEvent.Replicas
	return delta
}

func (s *replicaMemory) GetFirstSnapshotAfter(since time.Time) *ReplicaSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.events {
		if e.Timestamp.After(since) {
			return &e
		}
	}
	return nil
}
