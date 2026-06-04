package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
	bold   = color.New(color.Bold)
)

// Success prints a green ✓-prefixed message.
func Success(format string, a ...interface{}) {
	green.Fprint(os.Stdout, "✓ ")
	fmt.Fprintf(os.Stdout, format+"\n", a...)
}

// Warn prints a yellow ⚠-prefixed message.
func Warn(format string, a ...interface{}) {
	yellow.Fprint(os.Stdout, "⚠ ")
	fmt.Fprintf(os.Stdout, format+"\n", a...)
}

// Error prints a red ✗-prefixed message to stderr.
func Error(format string, a ...interface{}) {
	red.Fprint(os.Stderr, "✗ ")
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Bold prints a bold line.
func Bold(format string, a ...interface{}) {
	bold.Fprintf(os.Stdout, format+"\n", a...)
}

// Plain prints a normal line.
func Plain(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\n", a...)
}
