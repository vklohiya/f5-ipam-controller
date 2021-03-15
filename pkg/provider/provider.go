package provider

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/vklohiya/f5-ipam-controller/pkg/provider/sqlite"
	log "github.com/vklohiya/f5-ipam-controller/pkg/vlogger"
)

type IPAMProvider struct {
	store      *sqlite.DBStore
	ipamLabels map[string]bool
}

type Params struct {
	Range string
}

func NewProvider(params Params) *IPAMProvider {
	// IPRangeMap := `{"test":"172.16.1.1-172.16.1.5", "prod":"172.16.1.50-172.16.1.55"}`

	prov := &IPAMProvider{
		store:      sqlite.NewStore(),
		ipamLabels: make(map[string]bool),
	}
	if !prov.Init(params) {
		return nil
	}
	return prov
}

func (prov *IPAMProvider) Init(params Params) bool {
	ipRangeMap := make(map[string]string)
	err := json.Unmarshal([]byte(params.Range), &ipRangeMap)
	if err != nil {
		log.Fatal("[PROV] Invalid IP range provided")
		return false
	}

	for ipamLabel, ipRange := range ipRangeMap {
		ipRangeConfig := strings.Split(ipRange, "-")
		if len(ipRangeConfig) != 2 {
			return false
		}

		startIP := net.ParseIP(ipRangeConfig[0])
		if startIP == nil {
			return false
		}

		endIP := net.ParseIP(ipRangeConfig[1])
		if endIP == nil {
			return false
		}

		var ips []string
		for ; startIP.String() != endIP.String(); incIP(startIP) {
			ips = append(ips, startIP.String())
		}

		if len(ips) == 0 {
			return false
		}
		prov.ipamLabels[ipamLabel] = true
		prov.store.InsertIP(ips, ipamLabel)
	}
	prov.store.DisplayIPRecords()

	return true
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Creates an A record
func (prov *IPAMProvider) CreateARecord(hostname, ipAddr string) bool {
	prov.store.CreateARecord(hostname, ipAddr)
	log.Debugf("[PROV] Created 'A' Record. Host:%v, IP:%v", hostname, ipAddr)
	return true
}

// Deletes an A record and releases the IP address
func (prov *IPAMProvider) DeleteARecord(hostname, ipAddr string) {
	prov.store.DeleteARecord(hostname, ipAddr)
	log.Debugf("[PROV] Deleted 'A' Record. Host:%v, IP:%v", hostname, ipAddr)
}

func (prov *IPAMProvider) GetIPAddress(ipamLabel, hostname string) string {
	if _, ok := prov.ipamLabels[ipamLabel]; !ok {
		log.Debugf("[PROV] IPAM LABEL: %v Not Found", ipamLabel)
		return ""
	}
	return prov.store.GetIPAddress(ipamLabel, hostname)
}

// Gets and reserves the next available IP address
func (prov *IPAMProvider) GetNextAddr(ipamLabel string) string {
	if _, ok := prov.ipamLabels[ipamLabel]; !ok {
		log.Debugf("[PROV] Unsupported IPAM LABEL: %v", ipamLabel)
		return ""
	}
	return prov.store.AllocateIP(ipamLabel)
}

// Marks an IP address as allocated if it belongs to that IPAM LABEL
func (prov *IPAMProvider) AllocateIPAddress(ipamLabel, ipAddr string) bool {
	if _, ok := prov.ipamLabels[ipamLabel]; !ok {
		log.Debugf("[PROV] Unsupported IPAM LABEL: %v", ipamLabel)
		return false
	}

	return prov.store.MarkIPAsAllocated(ipamLabel, ipAddr)
}

// Releases an IP address
func (prov *IPAMProvider) ReleaseAddr(ipAddr string) {
	prov.store.ReleaseIP(ipAddr)
}
