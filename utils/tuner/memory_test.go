package tuner

import (
	"testing"
	"time"
)

func TestReplicaMemory(t *testing.T) {
	t.Run("test add and get events", func(t *testing.T) {
		rm := NewSimpleReplicaMemory(5, time.Minute)

		// Add events
		now := time.Now()
		events := []ReplicaSnapshot{
			{Timestamp: now.Add(-30 * time.Second), Replicas: 1},
			{Timestamp: now.Add(-20 * time.Second), Replicas: 2},
			{Timestamp: now.Add(-10 * time.Second), Replicas: 3},
		}

		for _, e := range events {
			rm.Add(e)
		}

		// Test GetMemorySince
		got := rm.GetMemorySince(now.Add(-25*time.Second), 0)
		if len(got) != 2 {
			t.Errorf("Expected 2 events, got %d", len(got))
		}
		if got[0].Replicas != 2 {
			t.Errorf("Expected replicas 2, got %d", got[0].Replicas)
		}
		if got[1].Replicas != 3 {
			t.Errorf("Expected replicas 3, got %d", got[1].Replicas)
		}
	})

	t.Run("test retention period", func(t *testing.T) {
		rm := NewSimpleReplicaMemory(5, time.Minute)
		now := time.Now()

		// Add old event beyond retention
		rm.Add(ReplicaSnapshot{
			Timestamp: now.Add(-2 * time.Minute),
			Replicas:  1,
		})

		// Add event within retention
		rm.Add(ReplicaSnapshot{
			Timestamp: now.Add(-30 * time.Second),
			Replicas:  2,
		})

		events := rm.GetMemorySince(now.Add(-3*time.Minute), 0)
		if len(events) != 1 {
			t.Errorf("Expected 1 event after retention period, got %d", len(events))
		}
	})

	t.Run("test max size", func(t *testing.T) {
		rm := NewSimpleReplicaMemory(3, time.Minute*10)
		now := time.Now()

		// Add more events than max size
		for i := 4; i >= 0; i-- {
			rm.Add(ReplicaSnapshot{
				Timestamp: now.Add(time.Duration(-i) * time.Second),
				Replicas:  int32(i),
			})
		}

		events := rm.GetMemorySince(now.Add(-time.Hour), 0)
		if len(events) != 3 {
			t.Errorf("Expected 3 events (max size), got %d", len(events))
		}

		// Check if we kept the most recent events
		expected := []int32{2, 1, 0}
		for i, e := range events {
			if e.Replicas != expected[i] {
				t.Errorf("Expected replicas %d at position %d, got %d", expected[i], i, e.Replicas)
			}
		}
	})

	t.Run("test expired events", func(t *testing.T) {
		rm := NewSimpleReplicaMemory(3, time.Minute*10)
		rm.Add(ReplicaSnapshot{Timestamp: time.Now().Add(-3 * time.Minute), Replicas: 3})
		rm.Add(ReplicaSnapshot{Timestamp: time.Now().Add(-2 * time.Minute), Replicas: 2})
		rm.Add(ReplicaSnapshot{Timestamp: time.Now().Add(-1 * time.Minute), Replicas: 1})

		events := rm.GetMemorySince(time.Now().Add(-2*time.Minute-time.Second), 0)
		if len(events) != 2 {
			t.Errorf("Expected 2 events, got %d", len(events))
		}
	})

	t.Run("test get delta since", func(t *testing.T) {
		rm := NewSimpleReplicaMemory(3, time.Minute*10)
		rm.Add(ReplicaSnapshot{Timestamp: time.Now().Add(-3 * time.Minute), Replicas: 3})
		rm.Add(ReplicaSnapshot{Timestamp: time.Now().Add(-2 * time.Minute), Replicas: 2})
		rm.Add(ReplicaSnapshot{Timestamp: time.Now().Add(-1 * time.Minute), Replicas: 1})

		delta := rm.GetDeltaSince(time.Now().Add(-2*time.Minute-time.Second), 0)
		if delta != -1 {
			t.Errorf("Expected delta -1, got %d", delta)
		}
	})

	t.Run("test get delta without enough events", func(t *testing.T) {
		rm := NewSimpleReplicaMemory(3, time.Minute*10)
		delta := rm.GetDeltaSince(time.Now(), 0)
		if delta != 0 {
			t.Errorf("Expected delta 0, got %d", delta)
		}
	})

	t.Run("test jitter toleration", func(t *testing.T) {
		rm := NewSimpleReplicaMemory(5, time.Minute)
		now := time.Now()

		rm.Add(ReplicaSnapshot{Timestamp: now.Add(-31 * time.Second), Replicas: 1})
		rm.Add(ReplicaSnapshot{Timestamp: now.Add(-29 * time.Second), Replicas: 2})

		// Without jitter
		events := rm.GetMemorySince(now.Add(-30*time.Second), 0)
		if len(events) != 1 {
			t.Errorf("Expected 1 event without jitter, got %d", len(events))
		}

		// With 2s jitter
		events = rm.GetMemorySince(now.Add(-30*time.Second), 2*time.Second)
		if len(events) != 2 {
			t.Errorf("Expected 2 events with jitter, got %d", len(events))
		}
	})
}

func BenchmarkReplicaMemory(b *testing.B) {
	rm := NewSimpleReplicaMemory(1000, time.Hour)
	now := time.Now()

	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rm.Add(ReplicaSnapshot{
				Timestamp: now.Add(time.Duration(i) * time.Second),
				Replicas:  int32(i),
			})
		}
	})

	b.Run("GetMemorySince", func(b *testing.B) {
		// Pre-populate with some events
		for i := 999; i >= 0; i-- {
			rm.Add(ReplicaSnapshot{
				Timestamp: now.Add(time.Duration(i) * time.Second),
				Replicas:  int32(i),
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rm.GetMemorySince(now, 0)
		}
	})

	b.Run("GetDeltaSince", func(b *testing.B) {
		// Pre-populate with some events
		for i := 999; i >= 0; i-- {
			rm.Add(ReplicaSnapshot{
				Timestamp: now.Add(time.Duration(i) * time.Second),
				Replicas:  int32(i),
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rm.GetDeltaSince(now, 0)
		}
	})
}
