package colors

// Red wraps the input string in ANSI red
func Red(s string) string {
	return "\x1b[31m" + string(s) + "\x1b[0m"
}

// Green wraps the input string in ANSI green
func Green(s string) string {
	return "\x1b[32m" + string(s) + "\x1b[0m"
}

// Yellow wraps the input string in ANSI yellow
func Yellow(s string) string {
	return "\x1b[33m" + string(s) + "\x1b[0m"
}
