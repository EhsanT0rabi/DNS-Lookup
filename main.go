// Help pages
// https://cabulous.medium.com/dns-message-how-to-read-query-and-response-message-cfebcb4fe817
// https://www.nullhardware.com/blog/dns-basics/

package main

import (
	"encoding/binary"
	"fmt"
	"github.com/urfave/cli"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Answer struct {
	name  string
	ip    net.IP
	ttl   uint32
	type_ string
	class string
}

func classParser(classNum uint8) string {
	classTypes := make(map[uint8]string)
	classTypes[1] = "IN"
	classTypes[2] = "CS"
	classTypes[3] = "CH"
	classTypes[4] = "HS"
	return classTypes[classNum]
}

func typeParser(typeNum uint8) string {
	dnsTypes := make(map[uint8]string)
	dnsTypes[1] = "A"
	dnsTypes[15] = "MX"
	dnsTypes[5] = "CNAME"
	dnsTypes[2] = "NS"
	return dnsTypes[typeNum]
}

func buildDNSQuery(domain string) ([]byte, error) {
	// Transaction ID: random 2 bytes
	var transactionID uint16 = uint16(0x0000 + rand.Intn(1000))
	flags := uint16(0x0100) // standard query, recursive
	qdCount := uint16(1)    // 1 question
	anCount, nsCount, arCount := uint16(0), uint16(0), uint16(0)

	// Header section
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], transactionID)
	binary.BigEndian.PutUint16(header[2:4], flags)
	binary.BigEndian.PutUint16(header[4:6], qdCount)
	binary.BigEndian.PutUint16(header[6:8], anCount)
	binary.BigEndian.PutUint16(header[8:10], nsCount)
	binary.BigEndian.PutUint16(header[10:12], arCount)

	// Question section
	question := make([]byte, 0)
	for _, label := range strings.Split(domain, ".") {
		question = append(question, byte(len(label)))
		question = append(question, []byte(label)...)
	}
	question = append(question, 0x00)       // end of domain name
	question = append(question, 0x00, 0x01) // QTYPE A
	question = append(question, 0x00, 0x01) // QCLASS IN

	// Combine header and question
	return append(header, question...), nil
}

func parseDNSResponse(response []byte, domain string) (Answer, error) {
	//fmt.Printf("respo:\t% x\n", response)
	ipStart := len(response) - 4
	ttlStart := len(response) - 10
	classStart := len(response) - 12
	typeStart := len(response) - 14
	ip := net.IPv4(response[ipStart], response[ipStart+1], response[ipStart+2], response[ipStart+3])
	ttl := binary.BigEndian.Uint32(response[ttlStart : ttlStart+4])
	typeNum := binary.BigEndian.Uint16(response[typeStart : typeStart+2])
	type_ := typeParser(uint8(typeNum))
	classNum := binary.BigEndian.Uint16(response[classStart : classStart+2])
	class := classParser(uint8(classNum))

	result := Answer{name: domain, ip: ip, ttl: ttl, type_: type_, class: class}
	return result, nil
}

func resolveDomain(domain string, dnsServer string) (Answer, error) {
	query, err := buildDNSQuery(domain)
	//fmt.Printf("query:\t% x\n", query)
	if err != nil {
		return Answer{}, err
	}

	conn, err := net.Dial("udp", dnsServer+":53")
	if err != nil {
		return Answer{}, err
	}
	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return Answer{}, err
	}
	if _, err := conn.Write(query); err != nil {
		return Answer{}, err
	}

	response := make([]byte, 512)
	read, err := conn.Read(response)
	if err != nil {
		return Answer{}, err
	}

	// Parse IP address from the response
	return parseDNSResponse(response[:read], domain)
}

func main() {
	app := NewCli()
	wg := sync.WaitGroup{}

	app.Action = func(c *cli.Context) {
		domains := strings.Split(c.String("domains"), ",")
		dnsServer := c.String("dns")
		fmt.Println("DNS Server: ", dnsServer)
		if len(os.Args) == 1 {
			fmt.Println("Usage: --domains domain1,domain2,...")
			return
		}
		wg.Add(len(domains))
		for _, s := range domains {
			s := s
			go func() {
				answer, err := resolveDomain(s, dnsServer)
				if err != nil {
					wg.Done()
					fmt.Println("Error:", err)
					return
				}
				fmt.Printf("The IP address for %s is %s \t TTL: %ds  \t type: %s \t class: %s \n", answer.name, answer.ip.String(), answer.ttl, answer.type_, answer.class)
				wg.Done()
			}()
		}
	}

	app.Run(os.Args)
	wg.Wait()
}
