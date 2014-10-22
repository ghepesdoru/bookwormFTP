package addr

import (
	"testing"
	"net"
	"fmt"
)

var (
	IPv4Sample = net.ParseIP("127.0.0.1")
	IPv6SampleSample = net.ParseIP("::1")
	Port = 49153
	PORTAddr = Addr{&IPv4Sample, Port, IPv4}
	PORTInput = "127,0,0,1,192,1"
	EPRTAddrV4 = Addr{&IPv4Sample, Port, IPv4}
	EPRTAddrV6 = Addr{&IPv6SampleSample, Port, IPv6}
	EPRTInputV4 = "|1|127.0.0.1|49153|"
	EPRTInputV6	= "|2|::1|49153|"
)

func TestPORT(t *testing.T) {
	addr := FromPortSpecifier(PORTInput)
	if addr.IP.String() != PORTAddr.IP.String() {
		t.Fatal("Invalid FromPortSpecifier ip parsing.", addr.IP)
	}
	if addr.Port != PORTAddr.Port {
		t.Fatal("Invalid FromPortSpecifier port parsing.", addr.Port)
	}
	if addr.IPFamily != PORTAddr.IPFamily {
		t.Fatal("Invalid FromPortSpecifier ip family parsing.", addr.IPFamily)
	}

	serialized := addr.ToPortSpecifier()
	if len(serialized) == 0 {
		t.Fatal("Invalid ToPortSpecifier serialization. Resulted in an empty string.")
	}

	if serialized != PORTInput {
		t.Fatal("Invalid ToPortSpecifier serialization.", serialized)
	}
}

func TestEPRT(t *testing.T) {
	addr4 := FromExtendedPortSpecifier(EPRTInputV4)
	addr6 := FromExtendedPortSpecifier(EPRTInputV6)

	if addr4.IP.String() != EPRTAddrV4.IP.String() {
		t.Fatal("Invalid FromExtendedPortSpecifier ip parsing (IPv4Sample).", addr4.IP)
	}
	if addr6.IP.String() != EPRTAddrV6.IP.String() {
		t.Fatal("Invalid FromExtendedPortSpecifier ip parsing (IPv6SampleSample).", addr6.IP)
	}

	if addr4.Port != EPRTAddrV4.Port {
		t.Fatal("Invalid FromExtendedPortSpecifier port parsing (IPv4Sample).", addr4.Port)
	}
	if addr6.Port != EPRTAddrV6.Port {
		t.Fatal("Invalid FromExtendedPortSpecifier port parsing (IPv6SampleSample).", addr6.Port)
	}

	if addr4.IPFamily != EPRTAddrV4.IPFamily {
		t.Fatal("Invalid FromExtendedPortSpecifier ip family parsing (IPv4Sample).", addr4.IPFamily)
	}
	if addr6.IPFamily != EPRTAddrV6.IPFamily {
		t.Fatal("Invalid FromExtendedPortSpecifier ip family parsing (IPv6SampleSample).", addr6.IPFamily)
	}

	serialized4 := addr4.ToExtendedPortSpecifier()
	serialized6 := addr6.ToExtendedPortSpecifier()

	if len(serialized4) == 0 {
		t.Fatal("Invalid FromExtendedPortSpecifier serialization. Resulted in an empty string.")
	}
	if len(serialized6) == 0 {
		t.Fatal("Invalid FromExtendedPortSpecifier serialization. Resulted in an empty string.")
	}

	if serialized4 != EPRTInputV4 {
		t.Fatal("Invalid FromExtendedPortSpecifier serialization.", serialized4)
	}
	if serialized6 != EPRTInputV6 {
		t.Fatal("Invalid FromExtendedPortSpecifier serialization.", serialized6)
	}
}

func TestToTCPAddr(t *testing.T) {
	addr4 := FromExtendedPortSpecifier(EPRTInputV4)
	addr6 := FromExtendedPortSpecifier(EPRTInputV6)

	tcpAddr4 := addr4.ToTCPAddr()
	tcpAddr6 := addr6.ToTCPAddr()

	if addr4.IP.String() != tcpAddr4.IP.String() {
		t.Fatal("Invalid ToTCPAddr ip translation (IPv4Sample).", tcpAddr4.IP)
	}
	if addr6.IP.String() != tcpAddr6.IP.String() {
		t.Fatal("Invalid ToTCPAddr ip translation (IPv6SampleSample).", tcpAddr6.IP)
	}

	if addr4.Port != tcpAddr4.Port {
		t.Fatal("Invalid ToTCPAddr port translation (IPv4Sample).", tcpAddr4.Port)
	}
	if addr6.Port != tcpAddr6.Port {
		t.Fatal("Invalid ToTCPAddr port translation (IPv6SampleSample).", tcpAddr6.Port)
	}
}

func TestFromConnection(t *testing.T) {
	addr4 := FromExtendedPortSpecifier(EPRTInputV4)
	addr6 := FromExtendedPortSpecifier(EPRTInputV6)
	tcpAddr4 := addr4.ToTCPAddr()
	tcpAddr6 := addr6.ToTCPAddr()

	l4, _ := net.ListenTCP("tcp", tcpAddr4)
	l6, _ := net.ListenTCP("tcp", tcpAddr6)
	defer l4.Close()
	defer l6.Close()

	go func(l4 net.Listener, l6 net.Listener) {
		for {
			_, _ = l4.Accept()
			_, _ = l6.Accept()
		}
	}(l4, l6)

	c4, _ := net.DialTCP("tcp", nil, tcpAddr4)
	c6, _ := net.DialTCP("tcp", nil, tcpAddr6)

	a4 := FromConnection(c4)
	a6 := FromConnection(c6)

	fmt.Println(a4, a6, addr4, addr6, tcpAddr4, tcpAddr6)

	if a4.IP.String() != addr4.IP.String() {
		t.Fatal("Invalid FromConnection ip extraction (IPv4Sample).", a4.IP)
	}
	if a6.IP.String() != addr6.IP.String() {
		t.Fatal("Invalid FromConnection ip extraction (IPv6SampleSample).", a6.IP)
	}

	/* Extraction from a connection will return an Addr with the the default 0 port */
	if a4.Port != 0 {
		t.Fatal("Invalid FromConnection port extraction (IPv4Sample).", a4.Port)
	}
	if a6.Port != 0 {
		t.Fatal("Invalid FromConnection port extraction (IPv6SampleSample).", a6.Port)
	}

	if a4.IPFamily != addr4.IPFamily {
		t.Fatal("Invalid FromConnection ip family extraction (IPv4Sample).", a4.IPFamily)
	}
	if a6.IPFamily != addr6.IPFamily {
		t.Fatal("Invalid FromConnection ip family extraction (IPv6SampleSample).", a6.IPFamily)
	}
}
