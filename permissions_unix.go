//go:build unix

package main

import "syscall"

// restrictFilePermissions makes every file this process creates (database,
// write-ahead log, sessions, WebKit storage) readable only by its owner.
func restrictFilePermissions() {
	syscall.Umask(0o077)
}
