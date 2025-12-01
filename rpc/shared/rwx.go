package shared

import (
	"sync"
	"time"
)

// RWXMutex is a writer-preferred RW lock with a minor respect for readers as well;
// preventing the starvation of readers and writers, whilst providing concurrent-safe
// access to resources.
// It's implemented with a timeout in mind such that if no reader's been inside for longer,
// than timeout, then after the current writer is done the batch of current waiting readers
// will be flewn inside.
type RWXMutex struct {
	mu sync.Mutex

	readersActive  int
	waitingReaders int
	writerActive   bool
	waitingWriters int

	readOk  *sync.Cond
	writeOk *sync.Cond

	timeout             time.Duration
	oldestReaderQueTime time.Time
	readerWindow        bool
}

func NewRWXMutex(timeout time.Duration) *RWXMutex {
	w := &RWXMutex{}
	w.readOk = sync.NewCond(&w.mu)
	w.writeOk = sync.NewCond(&w.mu)
	w.timeout = timeout
	return w
}

func (w *RWXMutex) RLock() {
	w.mu.Lock()
	// if there are no readers, the previous value has expired.
	if w.waitingReaders == 0 && !w.readerWindow {
		w.oldestReaderQueTime = time.Now()
	}
	w.waitingReaders++
	// If there is an active writer OR any writer waiting, readers must wait;
	//  unless there's a readerWindow going on.
	for w.writerActive || (w.waitingWriters > 0 && !w.readerWindow) {
		w.readOk.Wait()
	}
	w.waitingReaders--
	// when there are no more queued readers reset time
	if w.waitingReaders == 0 {
		w.oldestReaderQueTime = time.Time{}
	}
	w.readersActive++
	if w.waitingReaders > 0 && w.readerWindow {
		w.readOk.Signal()
	}
	w.mu.Unlock()
}

func (w *RWXMutex) RUnlock() {
	w.mu.Lock()
	w.readersActive--
	// If no more readers are waiting, wake one writer.
	if w.readersActive == 0 {
		w.readerWindow = false
		if w.waitingWriters > 0 {
			w.writeOk.Signal()
		}
	}
	w.mu.Unlock()
}

func (w *RWXMutex) Lock() {
	w.mu.Lock()
	w.waitingWriters++
	// wait until no active readers or writers
	for w.readersActive > 0 || w.writerActive || w.readerWindow {
		w.writeOk.Wait()
	}
	w.waitingWriters--
	w.writerActive = true
	w.mu.Unlock()
}

func (w *RWXMutex) Unlock() {
	w.mu.Lock()
	w.writerActive = false
	// prefer writers if any, unless timeout for readers being starved has been hit.
	if w.waitingReaders > 0 &&
		(w.waitingWriters == 0 ||
			(!w.oldestReaderQueTime.IsZero() && time.Since(w.oldestReaderQueTime) > w.timeout)) {
		w.readerWindow = true
		w.readOk.Signal()
	} else if w.waitingWriters > 0 {
		w.writeOk.Signal()
	}
	w.mu.Unlock()
}
