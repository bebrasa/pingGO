package main

import (
	"bytes"
	"flag"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"os"
	"time"
)

func main() {
	var (
		count    int
		size     int
		interval int
		timeout  int
	)

	flag.IntVar(&count, "c", 4, "number of echo requests to send")
	flag.IntVar(&size, "s", 56, "size of echo request payload")
	flag.IntVar(&interval, "i", 1000, "interval between echo requests (ms)")
	flag.IntVar(&timeout, "t", 1000, "timeout for each echo reply (ms)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: ping [options] <destination>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	dest := flag.Arg(0)
	ipAddr, err := net.ResolveIPAddr("ip4", dest)
	if err != nil {
		fmt.Printf("Failed to resolve %s: %v\n", dest, err)
		os.Exit(1)
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Printf("Failed to listen for ICMP packets: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("PING %s (%s): %d data bytes\n", dest, ipAddr.String(), size)

	for i := 0; i < count; i++ {
		id := os.Getpid() & 0xffff
		seq := i + 1

		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   id,
				Seq:  seq,
				Data: bytes.Repeat([]byte{byte(seq)}, size),
			},
		}

		msgBytes, err := msg.Marshal(nil)
		if err != nil {
			fmt.Printf("Failed to marshal ICMP message: %v\n", err)
			continue
		}

		start := time.Now()
		_, err = conn.WriteTo(msgBytes, ipAddr)
		if err != nil {
			fmt.Printf("Failed to send ICMP message: %v\n", err)
			continue
		}

		reply := make([]byte, 1500)
		err = conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
		if err != nil {
			fmt.Printf("Failed to set read deadline: %v\n", err)
			continue
		}

		n, peer, err := conn.ReadFrom(reply)
		if err != nil {
			fmt.Printf("Request timeout for icmp_seq %d\n", seq)
			continue
		}

		duration := time.Since(start)

		rm, err := icmp.ParseMessage(1, reply[:n])
		if err != nil {
			fmt.Printf("Failed to parse ICMP message: %v\n", err)
			continue
		}

		switch rm.Type {
		case ipv4.ICMPTypeEchoReply:
			echoReply := rm.Body.(*icmp.Echo)
			if echoReply.ID != id {
				fmt.Printf("Received ICMP echo reply with mismatched ID %d from %s\n", echoReply.ID, peer)
				continue
			}
			fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n", n, peer, echoReply.Seq, duration)
		default:
			fmt.Printf("Received unexpected ICMP message: %+v\n", rm)
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}
