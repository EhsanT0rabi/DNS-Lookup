// Help pages
// https://cabulous.medium.com/dns-message-how-to-read-query-and-response-message-cfebcb4fe817
// https://www.nullhardware.com/blog/dns-basics/

package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

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

func parseDNSResponse(response []byte) (string, error) {
	//fmt.Printf("respo:\t% x\n", response)
	ipStart := len(response) - 4

	ip := net.IPv4(response[ipStart], response[ipStart+1], response[ipStart+2], response[ipStart+3])
	return ip.String(), nil
}

func resolveDomain(domain string, dnsServer string) (string, error) {
	query, err := buildDNSQuery(domain)
	//fmt.Printf("query:\t% x\n", query)
	if err != nil {
		return "", err
	}

	conn, err := net.Dial("udp", dnsServer+":53")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return "", err
	}
	if _, err := conn.Write(query); err != nil {
		return "", err
	}

	response := make([]byte, 512)
	read, err := conn.Read(response)
	if err != nil {
		return "", err
	}

	// Parse IP address from the response
	return parseDNSResponse(response[:read])
}

func main() {
	dnsServer := "1.1.1.1"
	address := []string{"nic.ir", "sku.ac.ir", "google.com"}
	wg := sync.WaitGroup{}
	wg.Add(len(address))
	for _, s := range address {
		s := s
		go func() {
			ip, err := resolveDomain(s, dnsServer)
			if err != nil {
				wg.Done()
				fmt.Println("Error:", err)
				return
			}
			fmt.Printf("The IP address for %s is %s\n", s, ip)
			wg.Done()
		}()
	}
	wg.Wait()
}
