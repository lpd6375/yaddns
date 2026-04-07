//go:build linux
// +build linux

package main

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "log"
    "net"
    "github.com/cilium/ebpf"
    "github.com/cilium/ebpf/link"
    "github.com/cilium/ebpf/perf"
)

func run() error {
    spec, err := ebpf.LoadCollectionSpec("/usr/local/src/icmp_ddns_kern.o")
    if err != nil {
        return err
    }
    spec.Types = nil

    objs := struct {
        Program *ebpf.Program `ebpf:"icmp_ddns"`
        Events  *ebpf.Map     `ebpf:"events"`
    }{}

    if err := spec.LoadAndAssign(&objs, nil); err != nil {
        return err
    }

    ifIndex := getIfIndex(cfg.Interface)

    l, err := link.AttachXDP(link.XDPOptions{
        Program:   objs.Program,
        Interface: ifIndex,
        Flags:     link.XDPGenericMode,
    })
    if err != nil {
        return err
    }
    defer l.Close()

    rd, _ := perf.NewReader(objs.Events, 4096)
    defer rd.Close()

    log.Println("ICMP DDNS started")

    for {
        record, err := rd.Read()
        if err != nil {
            continue
        }

        var e Event
        if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &e); err != nil {
            continue
        }

        ip := net.IPv4(
            byte(e.SrcIP),
            byte(e.SrcIP>>8),
            byte(e.SrcIP>>16),
            byte(e.SrcIP>>24),
        ).String()

        if shouldUpdate(ip) {
            go updateCF(ip)
        }
    }
}
