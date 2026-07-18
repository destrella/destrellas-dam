//go:build darwin

package ui

func rutasFuentesSistemaPreferidas() []string {
	return []string{
		"/System/Library/Fonts/SFNS.ttf",
		"/System/Library/Fonts/SFNSItalic.ttf",
		"/System/Library/Fonts/Apple Color Emoji.ttc",
	}
}

func familiaFuenteSistemaPreferida() string {
	return "\"System Font\", \"Apple Color Emoji\""
}
