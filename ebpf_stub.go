//go:build !linux
// +build !linux

package main

import "log"

// run is a stub for non-Linux platforms so the admin server can run without eBPF.
func run() error {
    log.Println("eBPF disabled: running in admin-only mode (non-Linux build)")
    select {}
    return nil
}
