//go:build unix

package main

import "syscall"

// restrictFilePermissions makes every file this process creates (session file,
// safety snapshots, WebKit storage) readable only by its owner.
func restrictFilePermissions() {
	syscall.Umask(0o077)
}
