package addr

import (
	"net"
	"fmt"
	"strings"
	"strconv"
	"regexp"
)

/* Constants definition */
const (
	CONST_EPRTFormat = "|%d|%s|%d|"
	CONST_EPRTDelimiter = "|"
	CONST_IPv4Delimiter = "."
	CONST_PortDelimiter = ","
)

var (
	/* Matches a passive/port command address: ipv4,port (4 x 8bit + 2 x 8bit) */
	MatchHostAndPort 	= regexp.MustCompilePOSIX(`([0-9]{1,3}+,){5}+[0-9]{1,3}`)
	MatchEPSVResponse	= regexp.MustCompilePOSIX(`(\|+[^\|]*+\|+[^\|]*+\|[0-9]{1,5}+\|)`)
)

/* Addr type definition */
type Addr struct {
	IP			*net.IP
	Port		int
	IPFamily	int
}

/* Generates a PORT representation of the current Addr */
func (l *Addr) ToPortSpecifier() string {
	var parts []string

	/* Only for IPv4 */
	if l.IPFamily == 1 {
		parts = strings.Split(l.IP.String(), CONST_IPv4Delimiter)
		parts = append(parts, strconv.Itoa((l.Port >> 8) & 0xff))
		parts = append(parts, strconv.Itoa(l.Port & 0xff))

		return strings.Join(parts, CONST_PortDelimiter)
	}

	return ""
}

/* Generates a Addr from the received port PORT representation */
func FromPortSpecifier(dataPortMessage string) *Addr {
	var port int = -1
	ipAndPort := MatchHostAndPort.FindString(dataPortMessage)
	parts := strings.Split(ipAndPort, CONST_PortDelimiter)

	if len(parts) == 6 {
		p1, e1 := strconv.Atoi(parts[4])
		p2, e2 := strconv.Atoi(parts[5])

		if e1 == nil && e2 == nil {
			port = p1 * 256 + p2
		}
	}

	if port != -1 {
		ip := net.ParseIP(strings.Join(parts[:4], CONST_IPv4Delimiter))

		family := 1
		if ip.To4() == nil {
			family = 2
		}

		return &Addr{&ip, port, family}
	}

	return nil
}

/* Generate a EPRT compatible representation of the current address  */
func (l *Addr) ToExtendedPortSpecifier() string {
	return fmt.Sprintf(CONST_EPRTFormat, l.IPFamily, l.IP.String(), l.Port)
}

/* Generates a Addr from a EPRT compatible representation */
func FromExtendedPortSpecifier(epsv string) *Addr {
	input := MatchEPSVResponse.FindString(epsv)
	parts := strings.Split(input, CONST_EPRTDelimiter)

	if len(parts) == 5 {
		family, _ := strconv.Atoi(parts[1])
		port, _ := strconv.Atoi(parts[3])
		ip := net.ParseIP(parts[2])
		return &Addr{&ip, port, family}
	}

	return nil
}

/* Generates a Addr structure populated with the current connection's local host */
func FromConnectionLocal(c net.Conn) *Addr {
	var ipFamily int = 1 /* Assume IPv4 */

	addr := (c.LocalAddr()).(*net.TCPAddr)

	/* Determine the ip version in use */
	if addr.IP.To4() == nil {
		ipFamily = 2 /* IPv6 */
	}

	return &Addr{&addr.IP, 0, ipFamily}
}

/* Generates a Addr structure populated with the current connection's remote server */
func FromConnection(c net.Conn) *Addr {
	var ipFamily int = 1 /* Assume IPv4 */

	addr := (c.RemoteAddr()).(*net.TCPAddr)

	/* Determine the ip version in use */
	if addr.IP.To4() == nil {
		ipFamily = 2 /* IPv6 */
	}

	return &Addr{&addr.IP, 0, ipFamily}
}

/* Converts the current Address to a net.TCPAddr */
func (l *Addr) ToTCPAddr() *net.TCPAddr {
	return &net.TCPAddr{*l.IP, l.Port, ""}
}
