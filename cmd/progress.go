package cmd

import (
	"dotxt/task"
	"dotxt/terrors"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(incCmd, setCoundCmd)
	setIncCmdFlags()
	setCountCmdFlags()
}

var incCmd = &cobra.Command{
	Use:   "inc id [val=1] [--list==<todolist=todo>]",
	Short: "increment the count of a progress task",
	Long: `inc id [val=1] [--list==<todolist=todo>]
  increment the count of a progress task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		val := 1
		if len(args) >= 2 {
			val, err = strconv.Atoi(args[1])
			if err != nil {
				return err
			}
		}
		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}

		return loadFuncStoreFile(path, func() error {
			return task.IncrementProgressCount(id, path, val)
		})
	},
}

func setIncCmdFlags() {
	incCmd.Flags().String("list", "", "designate the target todolist")
}

var setCoundCmd = &cobra.Command{
	Use:   "setc id val [--list==<todolist=todo>]",
	Short: "set the count on a progress task",
	Long: `setc id val [--list==<todolist=todo>]
  set the count on a progress task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		if len(args) < 2 {
			return terrors.ErrorArgNotProvided("val")
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		val, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}

		return loadFuncStoreFile(path, func() error {
			return task.SetProgressCount(id, path, val)
		})
	},
}

func setCountCmdFlags() {
	setCoundCmd.Flags().String("list", "", "designate the target todolist")
}
