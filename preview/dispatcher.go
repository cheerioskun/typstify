package preview

import (
	"log/slog"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

type mainThreadFunc func()

// UIDispathcer is used to dispath webviews which is required
// to run in the main thread.
type dispatcher struct {
	funcQueue chan mainThreadFunc
}

func newDispathcer() *dispatcher {
	return &dispatcher{
		funcQueue: make(chan mainThreadFunc, 1),
	}
}

// run function in the main thread. fn may be blocking, so
// this prevents multiple webviews from running at the same
// time.
func (d *dispatcher) StartLoop() {
	for fn := range d.funcQueue {
		fn()
	}
}

func (d *dispatcher) Add(f mainThreadFunc) {
	select {
	case d.funcQueue <- f:
		slog.Info("request enqueued")
	default:
		// queue is full, skip
	}
}
