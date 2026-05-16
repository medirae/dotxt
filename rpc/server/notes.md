* server.go:Listen
```go
/*
	data:
		- check [<todolist>...]
			checkout the necesseties for timer management; maybe they should be requests too
		Write:
			- add <task> [--list=<todolist=todo>]
			- append <id> <task> [--list=<todolist=todo>]
			- deduplicate [--list==<todolist=todo>]
			- delete <id>... [--list==<todolist=todo>]
			- deprioritize <id>... [--list==<todolist=todo>]
			- done <id> [--list==<todolist=todo>]
			- increment id [val=1] [--list==<todolist=todo>]
			- migrate <from> [--list=<todolist=todo>]
			- move <from> <id> <to>
			- prepend <id> <task> [--list=<todolist=todo>]
			- prioritize <id> <priority> [--list=<todolist=todo>]
			- replace <id> <task> [--list=<todolist=todo>]
			- revert <id>... [--list==<todolist=todo>]
			- setc id val [--list==<todolist=todo>]
			- sort <todolist=todo>...
			- tc id [--list==<todolist=todo>]
		Read:
			- lsn id [--list==<todolist=todo>]
			- print <todolist=todo>...
			- print1 id [--list==<todolist=todo>]

	metadata:
		Read:
			- todo info
			- stats
			- config
		Write:
			- move/rename file
			- delete file
			- merge files
			- set config
*/
```

* scheduler.go
```go

/*
here's the thing:
there should be a type Request that represents something that the server has to

	somehow respond to.
	- requests from the cli/gui:
		- read from todo data
		- write to todo data
		- read metadata
		- write metadata
	- requests from watcher:
		- false positive events that the application triggered i.e. writing to a todo or metadata change
		- write to a file by user
		- deletion of file
		- creation of file in directory
	- requests made from the server: ?

sources of incoming requests include the fsnotify.Watcher goroutine and service Methods.

	every source should wrap the received data into a Request and send it into a channel
	that solely receives requests; let's call the channel the requests-ch

there should be a goroutine reading Requests from requests-ch and categorizing them.
these categories must not have any collisions with eachother in terms of data corruption.

	but since that would be impossible, there needs to be a locking category that serves as a
	category that everything in it will lock every other category; it has the upper hand and
	blocks all else. so when something that collides with all or nearly all other category
	requests, must come here and block all as to avoid data corruption.
	but for any other category - presuming the locking category is not blocking - the Requests
	of each one can only block the requests of that category.
	when a requests could not possibly corrupt any data then that should go to a free-for-all kind
	of category where when a request comes it is immediately served.
	categories: ?

each category must have its own scheduler. the scheduler for the category

	must receive the nearly-immediately requests and go over them and *sort* and store them.
	then it must go over the list of Requests as long as they are non-blocking Requests and
	dispatch them. when the scheduler meets a blocking Request and all other non-blocking
	Requests, if any, are after that, then the scheduler locks everything down until that
	blocking Request is processed; after which it must unlock. when it unlocks it should check,
	if a significant portion of time has passed, it shouldn't resume the current list of Requests,
	but rather it should again check whether there are any newer Requests that have come during
	this significant period of time, add them to the list, sort them, and start going over them from the beginning.
*/
```