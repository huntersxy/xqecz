package logx

import (
	"sort"
	"sync"
	"time"
)

type Entry struct {
	ID      uint64    `json:"id"`
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
	Line    string    `json:"line"`
}

type Buffer struct {
	mu      sync.RWMutex
	entries []Entry
	start   int
	nextID  uint64
}

func NewBuffer(capacity int) *Buffer {
	if capacity < 1 {
		capacity = defaultCapacity
	}
	return &Buffer{entries: make([]Entry, capacity)}
}

func (b *Buffer) Append(entry Entry) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	count := b.countLocked()
	b.nextID++
	entry.ID = b.nextID
	if entry.Time.IsZero() {
		entry.Time = time.Now()
	}
	idx := (b.start + count) % len(b.entries)
	if count == len(b.entries) {
		b.entries[idx] = entry
		b.start = (b.start + 1) % len(b.entries)
		return
	}
	b.entries[idx] = entry
}

func (b *Buffer) countLocked() int {
	if b.nextID < uint64(len(b.entries)) {
		return int(b.nextID)
	}
	return len(b.entries)
}

func (b *Buffer) Snapshot(after uint64, limit int) (entries []Entry, latest uint64, truncated bool) {
	if b == nil {
		return nil, 0, false
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	count := b.countLocked()
	latest = b.nextID
	if count == 0 {
		return []Entry{}, latest, false
	}
	firstID := latest - uint64(count) + 1
	if after > 0 && after+1 < firstID {
		truncated = true
		after = firstID - 1
	}
	startID := firstID
	if after > 0 {
		startID = after + 1
	}
	if startID > latest {
		return []Entry{}, latest, truncated
	}
	if limit <= 0 || limit > count {
		limit = count
	}
	if available := int(latest - startID + 1); available > limit {
		startID = latest - uint64(limit) + 1
	}
	entries = make([]Entry, 0, int(latest-startID+1))
	for id := startID; id <= latest; id++ {
		idx := (b.start + int(id-firstID)) % len(b.entries)
		entries = append(entries, b.entries[idx])
	}
	return entries, latest, truncated
}

func Levels(entries []Entry) []string {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		seen[entry.Level] = struct{}{}
	}
	levels := make([]string, 0, len(seen))
	for level := range seen {
		levels = append(levels, level)
	}
	sort.Strings(levels)
	return levels
}
