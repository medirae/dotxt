package shared

import (
	"dotxt/logging"
	"dotxt/utils"
	"os"
	"time"
)

type InternalEvent struct {
	size         int64     // before internal op
	modTime      time.Time // before internal op
	creationTime time.Time // of internal op
}

func (ie InternalEvent) IsValid() bool {
	return time.Since(ie.creationTime) <= internalEventTimeout
}

func (ie InternalEvent) HasChanged(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		logging.Logger.Warnf("internalEvent.hasNotChanged: failed to open path '%s': %w", path, err)
		return true
	}
	return ie.size != info.Size() || ie.modTime != info.ModTime()
}

func MarkInternalEvent(path string) {
	InternalEventsLock.Lock()
	info, err := os.Stat(path)
	if err != nil {
		InternalEventsLock.Unlock()
		logging.Logger.Errorf("MarkInternal: failed to open path '%s': %w", path, err)
		return
	}
	key := utils.GetFileCanonicalKey(path)
	_, ok := InternalEvents[key]
	if !ok {
		InternalEvents[key] = &InternalEvent{}
	}
	*InternalEvents[key] = InternalEvent{
		size:         info.Size(),
		modTime:      info.ModTime(),
		creationTime: time.Now(),
	}
	InternalEventsLock.Unlock()
}
