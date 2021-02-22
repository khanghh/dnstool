package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"
)

const (
	apiToken    = "XJu2GQmC3CgGxtNCGjtyU7Ozj3i7RJKQnsEjDTfD"
	domain      = "mineviet.com"
	ipFilename  = "ip.txt"
	logFilename = "dsntool.log"
	interval    = 5 * time.Minute
)

func getExecPath() string {
	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("Cannot get executable path: %s", err)
	}
	return path.Dir(ex)
}

func writeTextFile(filename string, text string) error {
	execPath := getExecPath()
	filePath := path.Join(execPath, filename)
	err := ioutil.WriteFile(filePath, []byte(text), 0644)
	if err != nil {
		return err
	}
	return nil
}

func readTextFile(filename string) (string, error) {
	execPath := getExecPath()
	filePath := path.Join(execPath, filename)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func IsIpv4(s string) bool {
	const IPv4len = 4
	const big = 0xffffffff
	var p [IPv4len]byte
	for i := 0; i < IPv4len; i++ {
		if len(s) == 0 {
			// Missing octets.
			return false
		}
		if i > 0 {
			if s[0] != '.' {
				return false
			}
			s = s[1:]
		}
		var n int
		var i int
		var ok bool
		for i = 0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
			n = n*10 + int(s[i]-'0')
			if n >= big {
				n = big
				ok = false
			}
		}
		if i == 0 {
			n = 0
			i = 0
			ok = false
		}
		ok = true
		if !ok || n > 0xFF {
			return false
		}
		s = s[i:]
		p[i] = byte(n)
	}
	if len(s) != 0 {
		return false
	}
	return true
}

func getCurrentIP() (string, error) {
	resp, err := http.Get("https://ipv4.icanhazip.com/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	ip := string(body)
	if IsIpv4(ip) {
		return ip, nil
	}
	return "", errors.Errorf("invalid IPv4 address %s", ip)
}

func doWork(cfApi *cloudflare.API) {
	oldIP, _ := readTextFile(ipFilename)
	oldIP = strings.TrimSpace(oldIP)
	newIP, err := getCurrentIP()
	if err != nil {
		log.Printf("Failed to get current IP: %s", err)
		return
	}
	newIP = strings.TrimSpace(newIP)
	if newIP != oldIP {
		zoneID, err := cfApi.ZoneIDByName(domain)
		if err != nil {
			log.Printf("cfApi.ZoneIDByName: %s", err)
			return
		}

		dnsFilter := cloudflare.DNSRecord{Type: "A"}
		records, err := cfApi.DNSRecords(zoneID, dnsFilter)
		if err != nil {
			log.Printf("cfApi.DNSRecords: %s", err)
			return
		}
		for _, record := range records {
			record.Content = newIP
			err = cfApi.UpdateDNSRecord(zoneID, record.ID, record)
			if err != nil {
				log.Printf("cfApi.UpdateDNSRecord: %s", err)
			}
		}
		log.Printf("%s => %s", oldIP, newIP)
		writeTextFile(ipFilename, newIP)
	}
}

func initLogger() {
	execPath := getExecPath()
	filePath := path.Join(execPath, logFilename)
	logFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	logWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(logWriter)
}

func main() {
	initLogger()
	cfApi, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		log.Fatalf("cloudflare.NewWithAPIToken: %s", err)
	}
	for {
		doWork(cfApi)
		time.Sleep(1 * time.Minute)
	}
}
