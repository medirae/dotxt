package server

import (
	"runtime"
	"time"
)

var (
	serverTimer         *time.Timer
	serverTimeout       time.Duration = 10 * time.Second
	requestTimeout      time.Duration
	requestAdmitTimeout time.Duration
	numWorkers                              = max(min(4*runtime.NumCPU(), 32), 2)
	requestsCh          chan *RequestHandle = make(chan *RequestHandle, 2*runtime.NumCPU())
)
