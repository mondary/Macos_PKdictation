//go:build !darwin

package main

import "errors"

func Run(hotkeyKeycode uint16, locale string) error {
	_ = hotkeyKeycode
	_ = locale
	return errors.New("cette application ne fonctionne que sur macOS (darwin)")
}

