package cmd

import (
	"dotxt/config"
	"dotxt/logging"
	"dotxt/terrors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const version = "0.0.0"

var rootCmd = &cobra.Command{
	Use:           "dotxt",
	Short:         fmt.Sprintf("dotxt %s: a text based todo list inspired by todotxt", version),
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.SetHelpTemplate(`
{{ with (or .Long .Short) }}{{ . | trimTrailingWhitespaces }}

{{ end}}Usage:{{if .Runnable}}
  {{ .UseLine }} [flags]{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath }} [command]{{end}}{{if gt (len .Aliases) 0 }}

Aliases:
  {{ .NameAndAliases }}{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{if .HasAvailableSubCommands}}}

Available Commands:
{{- range .Commands }}
  {{ rpad .NameAndAliases 20 }} {{ .Short }}
{{- end}}{{end}}{{if .HasAvailableFlags}}{{if not .Parent}}

Flags:
{{ .Flags.FlagUsages | trimTrailingWhitespaces }}{{else}}

{{ if .HasInheritedFlags }}Local {{end}}Flags:
{{ .LocalFlags.FlagUsages | trimTrailingWhitespaces }}{{if .HasInheritedFlags}}

Global Flags:
{{ .InheritedFlags.FlagUsages | trimTrailingWhitespaces }}{{end}}{{end}}{{end}}
`)
	cobra.OnInitialize(func() {
		arg, err := rootCmd.PersistentFlags().GetString("config")
		if err != nil {
			logging.Logger.Fatal(err)
		}
		if err := config.InitViper(arg); err != nil {
			logging.Logger.Fatal(err)
		}
		if lvl := viper.GetInt("logging.console-level"); -1 <= lvl && lvl <= 5 {
			logging.ConsoleLevel = min(logging.ConsoleLevel, viper.GetInt("logging.console-level"))
		} else {
			logging.Logger.Fatal(fmt.Errorf("%w: %w: flag 'clvl' must be between '-1' and '5' and not '%d'", terrors.ErrFlag, terrors.ErrValue, lvl))
		}

		if err := logging.InitFile(filepath.Join(config.ConfigPath(), "log")); err != nil {
			logging.Logger.Fatal(err)
		}
		logging.InitConsole(config.Quiet)
		logging.Initialize()
	})
	rootCmd.PersistentFlags().StringP("config", "c", "", "yaml config filepath")
	rootCmd.PersistentFlags().IntVar(&logging.ConsoleLevel, "clvl", 5, "console log -1 <= level <= 5")
	rootCmd.PersistentFlags().BoolVar(&config.Color, "color", false, "enable colored mode")
}

func Execute() error {
	return rootCmd.Execute()
}

func Silence() {
	rootCmd.SetOut(io.Discard)
}
