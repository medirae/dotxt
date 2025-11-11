I've written a filetext-based todolist in golang.
right now, there's a bunch of cobra commands in a `cmd` package that are being registered through main.go. the user can call the functions directly through this cli.
there's also a bash script I've written, which takes one command as arg and for the rest of the args that that command needs it uses rofi to capture user input. and I've bound each command through i3 bindings.
I've also created a conky widget that gets data by calling the print command of the cli.

this is extremely costly. per less than a second, every file of the todolist is being read, then parsed, then formatted, then printed out; like I said, extremely costly.

I'd like to make the application into client-server.
the server would run by the client when client needs something. when the client is done, the server would wait a designated amount of time for requests, and if that time is passed, the server has to tear itself down since it's not in use anymore.
when the server starts, it will read the files and parse the data and format the tasks and store the formattings. everything stays in memory hence.
the server must use grpc to communicate. some functions of its api that might change a value in the data, must block clients to avoid race conditions. others that are read-only can have racing conditions, but only when no write-only function is being run; if a write function is being run, read functions should be blocked as well until the writing is done. there must be no misjudgement in the stored and represented data.
the current commands can be kept in tact, and calling upon them implicitly means using the cli. but there should be other commands registered, one for a gui that I will later on get to, and another for running the server, which will be run by the cli and the future gui.

this is my current file structure:
```
/
/cmd: stores cli commands
/config: deals with configuration using viper
/pkg
/logging
/task: this is the core program where the magic happens; it has an api.go file that exposes the usages of the package to the rest of the codebase
/terrors: this is where I store my codebase custom defined errors
/utils: this is some utility functions used by the whole codebase
/main.go
```

obviously this has restructured.
decide the changes intially by yourself.
but some ideas:
	- there should be dedicated server and gui packages, the functions of which will be invoked by the cmd package
	- there should be a dedicated grpc package for the cli and gui packages to use; it will contain a bunch of utility-like functions that will help the cli and the gui packages as the client's of the server to call upon functions and receive data.
	- there's no need for a cli package since its functionalities are implied by the cmd package.
	- the utils, logging, terrors, and task package must be able to be used by the cmd, server, gprc, and gui packages.
	- it is likely that there will be multiple guis developed, one as a widget, the other as a means of getting user input; tell me which is better, having a dedicated package for each, or putting them in one general gui package since there will be things that both will use or something... do note that the gui libraries for each *might* vary.
	- the server package won't define excplicit functions to expose, it will only expose the functions already defined in the task package.

that's it for now.
