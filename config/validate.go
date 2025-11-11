package config

import (
	"dotxt/terrors"
	"dotxt/utils"
	"fmt"
	"unicode"

	"github.com/spf13/viper"
)

func validateConfig() []error {
	var errs []error
	// logging.*
	{
		if err := validateLogLevel("logging.console-level"); err != nil {
			errs = append(errs, err)
		}
		if err := validateLogLevel("logging.file-level"); err != nil {
			errs = append(errs, err)
		}
	}

	// print.*
	{
		// print.color-*
		{
			for _, key := range []string{
				"color-header", "color-default", "color-index", "color-burnt",
				"color-running-event-text", "color-running-event",
				"color-imminent-deadline", "color-date-due",
				"color-date-dead", "color-date-r", "color-every",
				"color-dead-relations", "color-collapsed",
				"color-hidden", "color-anti-priority", "color-urgent",
			} {
				if err := validateColor("print." + key); err != nil {
					errs = append(errs, err)
				}
			}
		}
		// print.hints.*
		{
			for _, key := range []string{
				"color-at", "color-plus", "color-tag",
				"color-exclamation", "color-question",
				"color-star", "color-ampersand",
			} {
				if err := validateColor("print.hints." + key); err != nil {
					errs = append(errs, err)
				}
			}
		}
		// print.quotes.*
		{
			for _, key := range []string{"double", "single", "backticks"} {
				if err := validateColor("print.quotes." + key); err != nil {
					errs = append(errs, err)
				}
			}
		}
		// print.ids.*
		{
			for _, key := range []string{"saturation", "lightness"} {
				if err := validateSL("print.ids." + key); err != nil {
					errs = append(errs, err)
				}
			}
			for _, key := range []string{"start-hue", "end-hue"} {
				if err := validateHue("print.ids." + key); err != nil {
					errs = append(errs, err)
				}
			}
		}
		// print.progress.*
		{
			for _, key := range []string{"count", "done-count", "unit", "header"} {
				if err := validateColor("print.progress." + key); err != nil {
					errs = append(errs, err)
				}
			}
			if err := validateTypeInt("print.progress.bartext-len"); err == nil {
				val := viper.GetInt("print.progress.bartext-len")
				if val < 5 || val > 50 {
					errs = append(errs, fmt.Errorf("%w: %w: value of 'print.progress.bartext-len' must be between 5 and 50 not '%d'", terrors.ErrConf, terrors.ErrValue, val))
				}
			} else {
				errs = append(errs, err)
			}
			// print.progress.percentage.*
			{
				for _, key := range []string{
					"start-saturation", "end-saturation",
					"start-lightness", "end-lightness",
				} {
					if err := validateSL("print.progress.percentage." + key); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
		// print.priority.*
		{
			for _, key := range []string{"saturation", "lightness"} {
				if err := validateSL("print.priority." + key); err != nil {
					errs = append(errs, err)
				}
			}
			for _, key := range []string{"start-hue", "end-hue"} {
				if err := validateHue("print.priority." + key); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	return errs
}

func validateHue(key string) error {
	if err := validateTypeInt(key); err != nil {
		return err
	}
	val := viper.GetInt(key)
	if val < 0 || val > 360 {
		return fmt.Errorf("%w: %w: value for config key '%s' must be between '0' and '360' and not '%d'", terrors.ErrConf, terrors.ErrValue, key, val)
	}
	return nil
}

func validateSL(key string) error {
	if err := validateTypeNumber(key); err != nil {
		return err
	}
	val := viper.GetFloat64(key)
	if val >= 1 || val < 0 {
		return fmt.Errorf("%w: %w: value for config key '%s' must be between '0.0' and '1.0' and not '%.2f'", terrors.ErrConf, terrors.ErrValue, key, val)
	}
	return nil
}

func validateColor(key string) error {
	if err := validateTypeString(key); err != nil {
		return err
	}
	color := viper.GetString(key)
	colorLen := utils.RuneCount(color)
	if colorLen != 7 {
		return fmt.Errorf("%w: %w: length of hex color must be '7' and not '%d'", terrors.ErrConf, terrors.ErrValue, colorLen)
	}
	if color[0] != '#' {
		return fmt.Errorf("%w: %w: hex color must start with '#' and not '%c'", terrors.ErrConf, terrors.ErrValue, color[0])
	}
	for _, char := range color[1:] {
		if !(unicode.IsDigit(char) || unicode.IsLetter(char)) {
			return fmt.Errorf("%w: %w: hex color must only consist of letters and digits not '%c'", terrors.ErrConf, terrors.ErrValue, char)
		}
	}
	return nil
}

func validateLogLevel(key string) error {
	if err := validateTypeInt(key); err != nil {
		return err
	}
	val := viper.GetInt(key)
	if val < -1 || val > 5 {
		return fmt.Errorf("%w: %w: config key '%s' must be between '-1' and '5' and not '%d'", terrors.ErrConf, terrors.ErrValue, key, val)
	}
	return nil
}

func validateTypeNumber(key string) error {
	raw := viper.Get(key)
	switch raw.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return nil
	default:
		return fmt.Errorf("%w: %w: config key '%s' must be of a numeric type not '%T'", terrors.ErrConf, terrors.ErrType, key, raw)
	}
}

func validateTypeInt(key string) error {
	raw := viper.Get(key)
	switch raw.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return nil
	default:
		return fmt.Errorf("%w: %w: config key '%s' must be of an int type not '%T'", terrors.ErrConf, terrors.ErrType, key, raw)
	}
}

func validateTypeString(key string) error {
	raw := viper.Get(key)
	switch raw.(type) {
	case string:
		return nil
	default:
		return fmt.Errorf("%w: %w: config key '%s' must be of type string not '%T'", terrors.ErrConf, terrors.ErrType, key, raw)
	}
}
