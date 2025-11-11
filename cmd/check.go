package cmd

import (
	"dotxt/task"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check [<todolist>...]",
	Short: "check for a variety of things to fix",
	Long: `check [<todolist>]
  if no arg is provided, the check is performed for all files.
  check for a variety of things to fix:
  	- recurrence of a task based on '$every'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var paths []string
		if len(args) < 1 {
			var err error
			paths, err = task.LsFiles()
			if err != nil {
				return err
			}
		} else {
			paths = args
		}
		for _, path := range paths {
			if err := loadFuncStoreFile(path, func() error {
				return task.CheckAndRecurTasks(path)
			}); err != nil {
				return err
			}
		}
		return nil
	},
}
