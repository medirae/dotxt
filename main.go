package main

import (
	"dotxt/cmd"
	"dotxt/config"
	"dotxt/logging"

	// "dotxt/rpc/client"
	// "dotxt/rpc/server"

	"os"
	"slices"
	"strconv"
	"strings"
)

func checkQuietFlag() {
	for ndx := 1; ndx < len(os.Args); ndx++ {
		arg := os.Args[ndx]
		if arg == "-q" || arg == "--quiet" || strings.HasPrefix(arg, "--quiet=") {
			val := true
			if strings.HasPrefix(arg, "--quiet=") {
				var err error
				val, err = strconv.ParseBool(strings.TrimPrefix(arg, "--quiet="))
				if err != nil {
					logging.Logger.Error("failed parsing %s: %w", arg, err)
					os.Exit(2)
				}
			}
			if val {
				config.Quiet = true
				cmd.Silence()
			}
			os.Args = slices.Delete(os.Args, ndx, ndx+1)
			break
		}
	}
}

func main() {
	// for ndx := 1; ndx < len(os.Args); ndx++ {
	// 	arg := os.Args[ndx]
	// 	switch arg {
	// 	case "server":
	// 		server.RunServer()
	// 	case "client":
	// 		fmt.Println(client.AddTask("testing this", "workdesk"))
	// 	default:
	// 		os.Exit(1)
	// 	}
	// }
	// os.Exit(0)

	checkQuietFlag()
	defer func() {
		if err := logging.Close(); err != nil {
			os.Exit(2)
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			logging.Logger.Fatalf("UNEXPECTED PANIC: %v\n%s", r)
		}
	}()
	if err := cmd.Execute(); err != nil {
		logging.Logger.Errorf("error running command: %v", err)
		os.Exit(1)
	}
}
