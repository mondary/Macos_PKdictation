package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	var hotkey string
	var locale string

	flag.StringVar(&hotkey, "hotkey", "fn", "Push-to-talk key: fn, f6, f7, f8, rshift, lshift, rctrl, lctrl, ropt, lopt, cmd, lcmd, rcmd (or a numeric macOS virtual keycode like 0x3F)")
	flag.StringVar(&locale, "locale", "fr-FR", "Speech recognition locale (e.g. fr-FR, en-US). Use \"system\" to use system locale.")
	flag.Parse()

	keycode, err := parseHotkey(hotkey)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if strings.EqualFold(locale, "system") {
		locale = ""
	}

	if err := Run(keycode, locale); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseHotkey(s string) (uint16, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "fn", "function":
		return 0x3F, nil
	case "f6":
		return 0x61, nil
	case "f7":
		return 0x62, nil
	case "f8":
		return 0x64, nil
	case "rshift":
		return 0x3C, nil
	case "lshift":
		return 0x38, nil
	case "rctrl":
		return 0x3E, nil
	case "lctrl":
		return 0x3B, nil
	case "ropt", "ralt":
		return 0x3D, nil
	case "lopt", "lalt":
		return 0x3A, nil
	case "cmd", "command", "lcmd":
		return 0x37, nil
	case "rcmd":
		return 0x36, nil
	}

	if strings.HasPrefix(s, "vk:") {
		s = strings.TrimPrefix(s, "vk:")
	}
	v, perr := strconv.ParseUint(s, 0, 16)
	if perr != nil {
		return 0, fmt.Errorf("hotkey invalide %q (ex: f6 ou 0x61)", s)
	}
	if v > 0xFFFF {
		return 0, fmt.Errorf("hotkey keycode trop grand: %d", v)
	}
	return uint16(v), nil
}
