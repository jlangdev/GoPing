//inspired by the following repos
//https://github.com/sparrc/go-ping/blob/master/ping.go
//https://gist.github.com/lmas/c13d1c9de3b2224f9c26435eb56e6ef3

package main

import (
    "time"
    "os"
    "log"
    "fmt"
    "net"
    "golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"strconv"

)
/*
    Config vars for determining protocol dependant values
    https://pkg.go.dev/golang.org/x/net/icmp?tab=doc
*/
const (
    ProtocolICMP = 1
    ProtocolIPv6ICMP = 58
)

var (
	ipv4Config = map[string]string{
        "ip": "ip4:icmp",
        "udp": "udp4",
        "source":"0.0.0.0",
    }
	ipv6Config = map[string]string{
        "ip": "ip6:ipv6-icmp",
         "udp": "udp6",
         "source":"0:0:0:0:0:0:0:0",
        }
)
/*
    DNS/IP resolution
*/
func ResolveIP(ippref *string, addr string)(*net.IPAddr, error){
    dst, err := net.ResolveIPAddr(*ippref, addr)
    if err != nil {
        if *ippref == "ip4" {
            log.Printf("Address %s could not be resolved with argument %s: Attempting to resolve as ip6 \n", addr, *ippref)
            *ippref = "ip6"
        } else if *ippref == "ip6" {
            log.Printf("Address %s could not be resolved with argument %s: Attempting to resolve as ip4 \n", addr, *ippref)
            *ippref = "ip4"
        }
        dst, err = ResolveIPMismatch(ippref, addr)
    }
    return dst, err
}

func ResolveIPMismatch(ippref *string, addr string)(*net.IPAddr, error){
    dst, err := net.ResolveIPAddr(*ippref, addr)
    if err != nil {
        return nil, err
    }
    log.Printf("Successfully resolved %s with %s: Continuing... \n", addr, *ippref)
    return dst, err
}

/*
    Config var selector based on protocol
*/
func ConfigureIP(ippref string)(string, string, icmp.Type, int){
    var protocol string
    var proto int
    var source string
    var typ icmp.Type
    if ippref == "ip4" {
        protocol = ipv4Config["ip"]
        source = ipv4Config["source"]
        typ = ipv4.ICMPTypeEcho
        proto = ProtocolICMP
	} else if ippref=="ip6" {
        protocol = ipv6Config["ip"]
        source = ipv6Config["source"]
        typ = ipv6.ICMPTypeEchoRequest
        proto = ProtocolIPv6ICMP
    } else {
        return protocol, source, nil, 0
    }
    return protocol, source, typ, proto
}

/*
    Establish Packet Connection
*/
func Connect(ippref string)(*icmp.PacketConn, int, icmp.Type, error){
    protocol, source, typ, proto := ConfigureIP(ippref)
    connection, err := icmp.ListenPacket(protocol, source)
    if err != nil {
        return nil, 0, nil, err
    }
    return connection, proto, typ, err
}

/*
    Initialize message with byte stream data and count
*/
func CreateMessage(typ icmp.Type, seq int)([]byte, error){
    m := icmp.Message{
		Type: typ,
		Code: 0,
        Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff,
			Seq: seq, 
            Data: []byte(""),
        },
    }
    b, err := m.Marshal(nil)
    return b, err
}

/*
    Send and receive ICMP echo
*/
func Exchange(connection *icmp.PacketConn, b []byte, dst *net.IPAddr)([] byte, int, time.Duration, net.Addr,error){
    /*
        Begint timer
        Send message
    */
    start := time.Now()
    n, err := connection.WriteTo(b, dst)
    if err != nil {
        return nil, 0, 0, nil, err
    } else if n != len(b) {
        return nil, 0, 0, nil, fmt.Errorf("got %v; want %v", n, len(b))
    }

    /*
        Await reply 
        Stop timer
    */
    reply := make([]byte, 1500)
    err = connection.SetReadDeadline(time.Now().Add(10 * time.Second))
    if err != nil {
        return nil, 0, 0, nil, err
    }
    n, peer, err := connection.ReadFrom(reply)
    if err != nil {
        return nil, 0, 0, nil, err
    }
    ttl := time.Since(start)
    return reply, n, ttl, peer, nil
}

/*
    Initialize echo
    Call helper functions
    Connect > Create Message > Exchange Message > Parse Message
*/
func Echo(dst *net.IPAddr, ippref string, seq int) (*net.IPAddr, time.Duration, error) {
    connection, proto, typ, err := Connect(ippref)
    
    defer connection.Close()

    b, err := CreateMessage(typ, seq)

    reply, n, ttl, peer, err := Exchange(connection, b, dst)

    rm, err := icmp.ParseMessage(proto, reply[:n])
    if err != nil {
        return dst, 0, err
    }
    if rm.Type == ipv4.ICMPTypeEchoReply || rm.Type == ipv6.ICMPTypeEchoReply || rm.Type == ipv6.ICMPTypeRouterAdvertisement {
        return dst, ttl, nil
	}else {
        return dst, 0, fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
    }
}

/*
    Function that drives echo loop
    iterate seq to keep track of how many messages sent
*/
func Driver(address string, maxttl time.Duration, ippref string){
    seq := 1

    dst, err := ResolveIP(&ippref, address)
    if err != nil {
        log.Printf("Unable to resolve %s (%s): %s\n", address, dst, err)
		return
    }
    
    for{
		dst, dur, err := Echo(dst, ippref, seq)
		if err != nil {
			log.Printf("Error pinging %s (%s): %s\nExiting: Try a different address.\n", address, dst, err)
			return
		}
		diff := dur - maxttl
		if int64(diff) > 0 {
			log.Printf("(%d) Pinging %s @ (%s): TTL: %s ----- Time Exceeded by %s \n", seq, address, dst, dur, diff)
		}else{
			log.Printf("(%d) Pinging %s @ (%s): TTL: %s\n", seq, address, dst, dur)
        }
        seq += 1
		time.Sleep(2 * time.Second)
	}
}

/*
    Initialize and assign vars to args
    Expecting [["address"], [number], ["'ip4' or 'ip6'"]]
    Call Driver and run loop
*/
func main() {

	args := os.Args[1:]

	address := args[0]
	ttlarg, err := strconv.Atoi(args[1])
    ippref := args[2]
    
	if err != nil {
		log.Printf("Invalid Argument %s: Please use a a number\nExiting...", args[1])
		return
    }
    if ippref != "ip4" && ippref != "ip6" {
		log.Printf("Invalid Argument %s: Use 'ip4' or 'ip6'\nExiting...", args[1])
		return
    }
    
	maxttl := time.Duration(ttlarg)*time.Millisecond
	Driver(address, maxttl, ippref)

}