package cmd

import (
	"dotxt/task"
	"dotxt/terrors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(printCmd, toggleCollapseCmd, print1)
	setPrintCmdFlags()
	setToggleCollapsedCmdFlags()
	setPrint1CmdFlags()
}

var printCmd = &cobra.Command{
	Use:   "print <todolist=todo>...",
	Short: "print tasks from lists",
	Long: `print <todolist=todo>...
  print tasks from lists`,
	RunE: func(cmd *cobra.Command, args []string) error {
		maxlen, err := cmd.Flags().GetInt("maxlen")
		if err != nil {
			return err
		}
		if maxlen > 300 {
			return fmt.Errorf("%w: %w: maxlen must be less than or equals to '%d' and not '%d'", terrors.ErrFlag, terrors.ErrValue, 300, maxlen)
		}
		minlen, err := cmd.Flags().GetInt("minlen")
		if err != nil {
			return err
		}
		if minlen < 50 {
			return fmt.Errorf("%w: %w: minlen must be greater than or equals to '%d' and not '%d'", terrors.ErrFlag, terrors.ErrValue, 50, minlen)
		}

		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}
		if len(args) < 1 {
			all = true
		}
		if all {
			var err error
			args, err = task.LsFiles()
			if err != nil {
				return err
			}
		}
		for _, arg := range args {
			if err := task.LoadFile(arg); err != nil {
				return err
			}
		}
		return task.PrintLists(args, maxlen, minlen)
	},
}

func setPrintCmdFlags() {
	printCmd.Flags().Bool("all", false, "print all lists")
	printCmd.Flags().Int("maxlen", 80, "maximum length")
	printCmd.Flags().Int("minlen", 80, "maximum length")
}

var toggleCollapseCmd = &cobra.Command{
	Use:   "tc id [--list==<todolist=todo>]",
	Short: "toggle the collapse/expanse of the children of a task",
	Long: `tc id [--list==<todolist=todo>]
  toggle the collapse/expanse of the children of a task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}

		return loadFuncStoreFile(path, func() error {
			return task.ToggleCollapsed(id, path)
		})
	},
}

func setToggleCollapsedCmdFlags() {
	toggleCollapseCmd.Flags().String("list", "", "designate the target todolist")
}

var print1 = &cobra.Command{
	Use:   "print1 id [--list==<todolist=todo>]",
	Short: "print a single task from list",
	Long: `print1 id [--list==<todolist=todo>]
  print a single task from list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		maxlen, err := cmd.Flags().GetInt("maxlen")
		if err != nil {
			return err
		}
		if maxlen < 20 || maxlen > 300 {
			return fmt.Errorf("%w: %w: maxlen must be  '50' <= maxlen <= '300' and not '%d'", terrors.ErrFlag, terrors.ErrValue, maxlen)
		}
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}

		if err := task.LoadFile(path); err != nil {
			return err
		}
		return task.PrintTask(id, path, maxlen)
	},
}

func setPrint1CmdFlags() {
	print1.Flags().String("list", "", "designate the target todolist")
	print1.Flags().Int("maxlen", 80, "maximum length")
}
