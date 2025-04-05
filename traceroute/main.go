package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	MaxHops      = 20
	Timeout      = time.Second * 2
	ICMPProtocol = 1
)

func traceroute(host string) {
	ipAddr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		fmt.Println("Ошибка resolve:", err)
		return
	}
	fmt.Printf("Traceroute to %s (%s), %d hops max\n", host, ipAddr.String(), MaxHops)

	for ttl := 1; ttl <= MaxHops; ttl++ {
		start := time.Now()

		conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		if err != nil {
			fmt.Println("Ошибка ListenPacket:", err)
			return
		}
		defer conn.Close()

		p := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  ttl,
				Data: []byte("PING"),
			},
		}
		msg, _ := p.Marshal(nil)

		conn.IPv4PacketConn().SetTTL(ttl)
		_ = conn.SetDeadline(time.Now().Add(Timeout))

		_, err = conn.WriteTo(msg, &net.IPAddr{IP: ipAddr.IP})
		if err != nil {
			fmt.Printf("%2d  * (write error)\n", ttl)
			continue
		}

		reply := make([]byte, 1500)
		n, peer, err := conn.ReadFrom(reply)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("%2d  * (timeout)\n", ttl)
			continue
		}

		rm, err := icmp.ParseMessage(ICMPProtocol, reply[:n])
		if err != nil {
			fmt.Printf("%2d  * (parse error)\n", ttl)
			continue
		}

		switch rm.Type {
		case ipv4.ICMPTypeTimeExceeded:
			fmt.Printf("%2d  %s  %.2fms\n", ttl, peer.String(), float64(duration.Microseconds())/1000)
		case ipv4.ICMPTypeEchoReply:
			fmt.Printf("%2d  %s  %.2fms (destination)\n", ttl, peer.String(), float64(duration.Microseconds())/1000)
			return
		default:
			fmt.Printf("%2d  %s  ?\n", ttl, peer.String())
		}
	}
}

func main() {
	host := "google.com"
	traceroute(host)
}
