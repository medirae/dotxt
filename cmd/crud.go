package cmd

import (
	"dotxt/task"
	"dotxt/terrors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(
		addCmd, delCmd, appendCmd,
		prependCmd, replaceCmd,
		deduplicateCmd, deprioritizeCmd,
		prioritizeCmd, doneCmd,
		revertCmd, moveCmd, migrateCmd,
		lsNCmd, sortCmd)
	setAddCmdFlags()
	setDelCmdFlags()
	setAppendCmdFlags()
	setPrependCmdFlags()
	setReplaceCmdFlags()
	setDeduplicateCmdFlags()
	setDeprioritizeCmdFlags()
	setPrioritizeCmdFlags()
	setRevertCmdFlags()
	setDoneCmdFlags()
	setMigrateCmdFlags()
	setlsNCmdFlags()
	setSortCmdFlags()
}

func loadFuncStoreFile(path string, f func() error) error {
	if err := task.LoadFile(path); err != nil {
		return err
	}
	if err := f(); err != nil {
		return err
	}
	return task.StoreFile(path)
}

func loadorcreateFuncStoreFile(path string, f func() error) error {
	if err := task.LoadOrCreateFile(path); err != nil {
		return err
	}
	if err := f(); err != nil {
		return err
	}
	return task.StoreFile(path)
}

func prepTodoListArg(cmd *cobra.Command) (string, error) {
	path, err := cmd.Flags().GetString("list")
	if err != nil {
		return "", nil
	}
	if strings.TrimSpace(path) == "" {
		return task.DefaultTodo, nil
	}
	return path, nil
}

var addCmd = &cobra.Command{
	Use:   "add <task> [--list=<todolist=todo>]",
	Short: "add task",
	Long: `add <task> [--list=<todolist=todo>]
  adds task to todolist`,
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := strings.Join(args, " ")
		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}
		return loadorcreateFuncStoreFile(path, func() error {
			return task.AddTaskFromStr(arg, path)
		})
	},
}

func setAddCmdFlags() {
	addCmd.Flags().String("list", "", "designate the target todolist")
}

var delCmd = &cobra.Command{
	Use:   "del <id>... [--list==<todolist=todo>]",
	Short: "delete task",
	Long: `del|rm <id>... [--list==<todolist=todo>]
  removes task from todolist`,
	Aliases: []string{"rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		var ids []int
		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil {
				return terrors.ErrorArgParse("id", err)
			}
			ids = append(ids, num)
		}

		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.DeleteTasks(ids, path)
		})
	},
}

func setDelCmdFlags() {
	delCmd.Flags().String("list", "", "designate the target todolist")
}

var appendCmd = &cobra.Command{
	Use:   "app <id> <task> [--list=<todolist=todo>]",
	Short: "append to task",
	Long: `append|app <task> [--list=<todolist=todo>],
  appends text to the end of the designated task`,
	Aliases: []string{"append"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrorArgNotProvided("id")
		}
		if len(args) < 2 {
			return terrors.ErrEmptyText
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return terrors.ErrorArgParse("id", err)
		}
		text := strings.Join(args[1:], " ")
		if strings.TrimSpace(text) == "" {
			return terrors.ErrEmptyText
		}

		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.AppendToTask(id, text, path)
		})
	},
}

func setAppendCmdFlags() {
	appendCmd.Flags().String("list", "", "designate the target todolist")
}

var prependCmd = &cobra.Command{
	Use:   "prepend <id> <task> [--list=<todolist=todo>]",
	Short: "prepend to task",
	Long: `prepend|prep <task> [--list=<todolist=todo>],
  prepends text to the end of the designated task`,
	Aliases: []string{"prep"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrorArgNotProvided("id")
		}
		if len(args) < 2 {
			return terrors.ErrEmptyText
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return terrors.ErrorArgParse("id", err)
		}
		text := strings.Join(args[1:], " ")
		if strings.TrimSpace(text) == "" {
			return terrors.ErrEmptyText
		}

		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.PrependToTask(id, text, path)
		})
	},
}

func setPrependCmdFlags() {
	prependCmd.Flags().String("list", "", "designate the target todolist")
}

var replaceCmd = &cobra.Command{
	Use:   "replace <id> <task> [--list=<todolist=todo>]",
	Short: "replace line with a new task",
	Long: `replace|update <id> <task> [--list=<todolist=todo>]
  replace line with a new task`,
	Aliases: []string{"update"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrorArgNotProvided("id")
		}
		if len(args) < 2 {
			return terrors.ErrEmptyText
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}
		text := strings.Join(args[1:], " ")
		return loadFuncStoreFile(path, func() error {
			return task.ReplaceTask(id, text, path)
		})
	},
}

func setReplaceCmdFlags() {
	replaceCmd.Flags().String("list", "", "designate the target todolist")
}

var deduplicateCmd = &cobra.Command{
	Use:   "deduplicate [--list==<todolist=todo>]",
	Short: "deduplicate list",
	Long: `deduplicate [--list==<todolist=todo>]
  removes duplicated lines from a list. repetetive spaces and meta variables are ignored.`,
	Aliases: []string{"dedup"},
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.DeduplicateList(path)
		})
	},
}

func setDeduplicateCmdFlags() {
	deduplicateCmd.Flags().String("list", "", "designate the target todolist")
}

var deprioritizeCmd = &cobra.Command{
	Use:   "depri <id>... [--list==<todolist=todo>]",
	Short: "deprioritize task",
	Long: `depri|dp <id>... [--list==<todolist=todo>]
  deprioritizes task(s) (removes priority) from list`,
	Aliases: []string{"dp"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrorArgNotProvided("id")
		}
		var ids []int
		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil {
				return terrors.ErrorArgParse("id", err)
			}
			ids = append(ids, num)
		}

		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			for _, id := range ids {
				if err = task.DeprioritizeTask(id, path); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func setDeprioritizeCmdFlags() {
	deprioritizeCmd.Flags().String("list", "", "designate the target todolist")
}

var prioritizeCmd = &cobra.Command{
	Use:   "pri <id> <priority> [--list=<todolist=todo>]",
	Short: "prioritize task",
	Long: `pri|p <id> <priority> [--list=<todolist=todo>]
  prioritizes task given the id and a priority text in a list`,
	Aliases: []string{"p"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrorArgNotProvided("id")
		}
		if len(args) < 2 {
			return terrors.ErrorArgNotProvided("priority")
		}
		if len(args) > 2 || strings.IndexFunc(args[1], unicode.IsSpace) != -1 {
			return fmt.Errorf("%w: %w: priority cannot contain spaces: '%s'", terrors.ErrArg, terrors.ErrValue, strings.Join(args[1:], " "))
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
			return task.PrioritizeTask(id, args[1], path)
		})
	},
}

func setPrioritizeCmdFlags() {
	prioritizeCmd.Flags().String("list", "", "designate the target todolist")
}

var doneCmd = &cobra.Command{
	Use:   "done <id> [--list==<todolist=todo>]",
	Short: "finish and move task",
	Long: `do|done <id> [--list==<todolist=todo>]
  finish task`,
	Aliases: []string{"do"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrorArgNotProvided("id")
		}
		var ids []int
		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil {
				return terrors.ErrorArgParse("id", err)
			}
			ids = append(ids, num)
		}
		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.DoneTask(ids, path)
		})
	},
}

func setDoneCmdFlags() {
	doneCmd.Flags().String("list", "", "designate the target todolist")
}

var revertCmd = &cobra.Command{
	Use:   "revert <id>... [--list==<todolist=todo>]",
	Short: "revert tasks from done to list",
	Long: `revert <id>... [--list==<todolist=todo>]
  reverts tasks from done to list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		var ids []int
		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil {
				return terrors.ErrorArgParse("id", err)
			}
			ids = append(ids, num)
		}

		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.RevertTask(ids, path)
		})
	},
}

func setRevertCmdFlags() {
	revertCmd.Flags().String("list", "", "designate the target todolist")
}

var moveCmd = &cobra.Command{
	Use:   "move <from> <id> <to>",
	Short: "move task around",
	Long: `move|mv <from> <id> <to>
  move task to another list`,
	Aliases: []string{"mv"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrorArgNotProvided("from")
		}
		if len(args) < 2 {
			return terrors.ErrorArgNotProvided("id")
		}
		if len(args) < 3 {
			return terrors.ErrorArgNotProvided("to")
		}

		from, idString, to := args[0], args[1], args[2]
		if err := task.CheckFileExistence(from); err != nil {
			return err
		}
		id, err := strconv.Atoi(idString)
		if err != nil {
			return err
		}

		if err = task.LoadFile(from); err != nil {
			return err
		}
		if err = task.LoadOrCreateFile(to); err != nil {
			return err
		}
		if err = task.MoveTask(from, id, to); err != nil {
			return err
		}
		if err = task.StoreFile(to); err != nil {
			return err
		}
		if err = task.StoreFile(from); err != nil {
			return err
		}
		return nil
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate <from> [--list=<todolist=todo>]",
	Short: "migrate tasks from a given file",
	Long: `migrate <from> [--list=<todolist=todo>]
  migrate tasks from a given file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		from := args[0]
		to, err := cmd.Flags().GetString("to")
		if err != nil {
			return err
		}
		if strings.TrimSpace(to) == "" {
			to = filepath.Base(from)
		}
		return loadorcreateFuncStoreFile(to, func() error {
			return task.MigrateTasks(from, to)
		})
	},
}

func setMigrateCmdFlags() {
	migrateCmd.Flags().String("list", "", "designate the target todolist")
}

var lsNCmd = &cobra.Command{
	Use:   "lsn id [--list==<todolist=todo>]",
	Short: "print a single task from list",
	Long: `lsn id [--list==<todolist=todo>]
  print a single task from list`,
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

		if err := task.LoadFile(path); err != nil {
			return err
		}
		return task.OutputTask(id, path)
	},
}

func setlsNCmdFlags() {
	lsNCmd.Flags().String("list", "", "designate the target todolist")
}

var sortCmd = &cobra.Command{
	Use:   "sort <todolist=todo>...",
	Short: "sort the tasks of the list in-place",
	Long: `sort <todolist=todo>...
  sort the tasks of the list in-place`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
		for _, path := range args {
			if err := loadFuncStoreFile(path, func() error {
				return task.SortList(path)
			}); err != nil {
				return err
			}
		}
		return nil
	},
}

func setSortCmdFlags() {
	sortCmd.Flags().Bool("all", false, "sort all tasks")
}
