optimizations:
	- there should be a maximum number of files that can be opened at once
		use a custom implemented lru cache to gatekeep the file descriptors that can be inside the cache.
		the cache has a maximum size, say 128 or 256.
		when a file is done being open it should be released from the cache
		the cache should be protected via a mutex so it's concurrent-safe
		this should be integral within the file package
		all functions of the file package must abide by it
	- the functions of file package must use bufio.Reader and bufio.Writer
		after writing, sync should be called to ensure writes
	- conccurent jobs must be throttled using a sync.Pool.
		this could be applied to the output of Dispatcher
		so that the pool is in charge of running the actual goroutines
		advised number of workers is 8-32
	- there should be per-file locks to avoid data corruption
	- a global exclusive lock is too costly,
		instead it's more efficient to mimic its behavior
		by locking all the per-file locks in sorted order, and unlock them in reverse order
	- file writes, file creates, etc should be done atomically
		the file should be written in /tmp/dotxt-*/tmp-*
		and then renamed to where it should be
	- file writes must use fsync afterwards to ensure contents are flushed down the file
	- when it comes to searching, a reverse index must be built for the fuzzy finder
	- the watcher must have a event web so as to capture a bunch of
		events together when they're happening so close to eachother
		and on top of that web, there should be an expectation
		mechanism that allows events caused by internal file I/O
		to be handled properly
