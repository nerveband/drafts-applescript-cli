//go:build !darwin

package main

func stdinIsTerminal() bool { return false }
