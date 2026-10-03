package commands

import (
	"encoding/binary"
	"fmt"
	"lever/engine"
	"lever/utils"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var icmpDLL = windows.NewLazySystemDLL("iphlpapi.dll")
var icmpCreateFile = icmpDLL.NewProc("IcmpCreateFile")
var icmpSendEcho = icmpDLL.NewProc("IcmpSendEcho")
var icmpCloseHandle = icmpDLL.NewProc("IcmpCloseHandle")

func NewPingCmd() *engine.Command {
	var count int
	var interval float64

	cmd := &engine.Command{
		Use:   "ping [hostname/IP]",
		Short: "Linux-style infinite loop ICMP network latency monitor",
		Args:  utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {
			target := args[0]
			intervalDuration := time.Duration(interval * float64(time.Second))
			if intervalDuration <= 0 {
				return fmt.Errorf("ping error: interval must be positive")
			}
			if count < 0 {
				return fmt.Errorf("ping error: count must not be negative")
			}

			dstIP, err := net.ResolveIPAddr("ip4", target)
			if err != nil {
				return fmt.Errorf("ping error: unknown host '%s': %v", target, err)
			}
			handle, _, callErr := icmpCreateFile.Call()
			if handle == ^uintptr(0) {
				return fmt.Errorf("ping error: cannot open ICMP handle: %w", callErr)
			}
			defer icmpCloseHandle.Call(handle)

			fmt.Fprintf(os.Stdout, "PING %s (%s): 56 data bytes\n", target, dstIP.String())

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt)
			defer signal.Stop(sigChan)

			ticker := time.NewTicker(intervalDuration)
			defer ticker.Stop()

			var sent, received int64
			sequence := 0

			for {
				select {
				case <-sigChan:
					printPingSummary(target, sent, received)
					return nil
				case <-ticker.C:
					sequence++
					sent++

					ok, rtt, err := sendICMPEcho(handle, dstIP.IP)
					if err != nil {
						return fmt.Errorf("ping error: cannot send ICMP echo: %w", err)
					}
					if ok {
						received++
						fmt.Printf("56 bytes from %s: icmp_seq=%d time=%d ms\n", dstIP.String(), sequence, rtt)
					} else {
						fmt.Printf("No ICMP echo reply for icmp_seq %d\n", sequence)
					}

					if count > 0 && sequence >= count {
						printPingSummary(target, sent, received)
						if received == 0 {
							return fmt.Errorf("ping error: no ICMP echo replies received")
						}
						return nil
					}
				}
			}
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&count, "count", "c", 0, "stop after sending count packets")
	flags.Float64VarP(&interval, "interval", "i", 1.0, "seconds between sending each packet")
	return cmd
}

func printPingSummary(target string, sent, received int64) {
	loss := 0.0
	if sent > 0 {
		loss = float64(sent-received) / float64(sent) * 100.0
	}
	fmt.Printf("\n--- %s ping statistics ---\n", target)
	fmt.Printf("%d packets transmitted, %d packets received, %.1f%% packet loss\n", sent, received, loss)
}

func sendICMPEcho(handle uintptr, ip net.IP) (bool, uint32, error) {
	address := ip.To4()
	if address == nil {
		return false, 0, fmt.Errorf("IPv4 address required")
	}
	request := make([]byte, 56)
	// ICMP_ECHO_REPLY is 40 bytes on Windows; include payload and error space.
	reply := make([]byte, 40+len(request)+8)
	count, _, callErr := icmpSendEcho.Call(
		handle,
		uintptr(binary.LittleEndian.Uint32(address)),
		uintptr(unsafe.Pointer(&request[0])),
		uintptr(len(request)),
		0,
		uintptr(unsafe.Pointer(&reply[0])),
		uintptr(len(reply)),
		2000,
	)
	runtime.KeepAlive(request)
	runtime.KeepAlive(reply)
	if count == 0 {
		if code, ok := callErr.(syscall.Errno); ok && code >= 11000 && code < 11100 {
			return false, 0, nil
		}
		if callErr != nil && callErr != windows.Errno(0) {
			return false, 0, callErr
		}
		return false, 0, nil
	}
	status := binary.LittleEndian.Uint32(reply[4:8])
	roundTripMs := binary.LittleEndian.Uint32(reply[8:12])
	return status == 0, roundTripMs, nil
}
