package config

import (
	"dotxt/logging"
	"html/template"
	"strings"
)

var colors = map[string]string{
	"black":        "#000000",
	"blue":         "#4895bf",
	"blue-light":   "#6CC0FC",
	"blue-sky":     "#90fcf5",
	"brown":        "#c48660",
	"cyan":         "#05FCC6",
	"cyan-cool":    "#66D9EF",
	"cyan-light":   "#9FFCF3",
	"cyan-pale":    "#c6eceb",
	"default":      "#DEF4ED",
	"gold-light":   "#ffec99",
	"green":        "#21FC00",
	"green-bright": "#A6E22E",
	"green-light":  "#78FC76",
	"grey":         "#696969",
	"grey-dark":    "#4C4C4C",
	"grey-light":   "#919191",
	"jade-light":   "#6aa88f",
	"orange-light": "#e88761",
	"orange-warm":  "#FD971F",
	"pink-light":   "#fc8ae1",
	"purple":       "#9e64ea",
	"purple-dark":  "#6b5f73",
	"purple-light": "#d994fc",
	"red":          "#B61C1C",
	"red-dark":     "#fb4141",
	"red-light":    "#FC7876",
	"white":        "#FFFFFF",
	"white-dark":   "#c4c4c4",
	"yellow":       "#FCFC64",
	"yellow-dark":  "#b2bc45",
	"yellow-light": "#f4f07f",
	"yellow-pale":  "#f9f7b9",
}

var DefaultConfig string = `
[logging]
console-level = 5
file-level    = -1

[print]
color-header             = '{{ index .Colors "red-light" }}'
color-default            = '{{ index .Colors "default" }}'
color-index              = '{{ index .Colors "grey-light" }}'
color-burnt              = '{{ index .Colors "grey-dark" }}'
color-running-event-text = '{{ index .Colors "yellow-pale" }}'
color-running-event      = '{{ index .Colors "gold-light" }}'
color-imminent-deadline  = '{{ index .Colors "red-dark" }}'
color-date-due           = '{{ index .Colors "orange-light" }}'
color-date-end           = '{{ index .Colors "orange-light" }}'
color-date-dead          = '{{ index .Colors "orange-light" }}'
color-date-r             = '{{ index .Colors "jade-light" }}'
color-urgent  			 = '{{ index .Colors "red-light" }}'
color-mit				 = '{{ index .Colors "blue-sky" }}'
color-every              = '{{ index .Colors "yellow-light" }}'
color-dead-relations     = '{{ index .Colors "grey" }}'
color-collapsed          = '{{ index .Colors "orange-light" }}'
color-focus  			 = '{{ index .Colors "red-light" }}'
color-hidden			 = '{{ index .Colors "grey-light" }}'
color-anti-priority      = '{{ index .Colors "grey-light" }}'

[print.hints]
color-at          = '{{ index .Colors "blue" }}'
color-plus        = '{{ index .Colors "jade-light" }}'
color-tag         = '{{ index .Colors "pink-light" }}'
color-exclamation = '{{ index .Colors "red-light" }}'
color-question    = '{{ index .Colors "blue-light" }}'
color-star        = '{{ index .Colors "yellow-light" }}'
color-ampersand   = '{{ index .Colors "brown" }}'

[print.quotes]
double    = '{{ index .Colors "green-bright" }}'
single    = '{{ index .Colors "orange-warm" }}'
backticks = '{{ index .Colors "cyan-cool" }}'

[print.ids]
saturation = 0.35
lightness  = 0.55
start-hue  = 30
end-hue    = 210

[print.progress]
count        = '{{ index .Colors "default" }}'
done-count   = '{{ index .Colors "grey-light" }}'
unit         = '{{ index .Colors "default" }}'
bartext-len  = 10
header       = '{{ index .Colors "purple-dark" }}'

[print.progress.percentage]
start-saturation = 0.45
end-saturation   = 0.7
start-lightness  = 0.47
end-lightness    = 0.55

[print.priority]
saturation  = 0.7
lightness   = 0.6
start-hue   = 0
end-hue     = 360
`

func init() {
	tmpl, err := template.New("config").Parse(DefaultConfig)
	if err != nil {
		logging.Logger.Fatalf("error parsing template: %v", err)
	}

	var str strings.Builder
	err = tmpl.Execute(&str, struct{ Colors map[string]string }{colors})
	if err != nil {
		logging.Logger.Fatalf("error executing template: %v", err)
	}
	DefaultConfig = str.String()
}
