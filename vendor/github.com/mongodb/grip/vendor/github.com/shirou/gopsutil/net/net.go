package net

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/internal/common"
)

var invoke common.Invoker

func init() {
	invoke = common.Invoke{}
}

type IOCountersStat struct {
	Name        string `json:"name" bson:"name,omitempty"`               // interface name
	BytesSent   uint64 `json:"bytesSent" bson:"bytesSent,omitempty"`     // number of bytes sent
	BytesRecv   uint64 `json:"bytesRecv" bson:"bytesRecv,omitempty"`     // number of bytes received
	PacketsSent uint64 `json:"packetsSent" bson:"packetsSent,omitempty"` // number of packets sent
	PacketsRecv uint64 `json:"packetsRecv" bson:"packetsRecv,omitempty"` // number of packets received
	Errin       uint64 `json:"errin" bson:"errin,omitempty"`             // total number of errors while receiving
	Errout      uint64 `json:"errout" bson:"errout,omitempty"`           // total number of errors while sending
	Dropin      uint64 `json:"dropin" bson:"dropin,omitempty"`           // total number of incoming packets which were dropped
	Dropout     uint64 `json:"dropout" bson:"dropout,omitempty"`         // total number of outgoing packets which were dropped (always 0 on OSX and BSD)
	Fifoin      uint64 `json:"fifoin" bson:"fifoin,omitempty"`           // total number of FIFO buffers errors while receiving
	Fifoout     uint64 `json:"fifoout" bson:"fifoout,omitempty"`         // total number of FIFO buffers errors while sending

}

// Addr is implemented compatibility to psutil
type Addr struct {
	IP   string `json:"ip" bson:"ip,omitempty"`
	Port uint32 `json:"port" bson:"port,omitempty"`
}

type ConnectionStat struct {
	Fd     uint32  `json:"fd" bson:"fd,omitempty"`
	Family uint32  `json:"family" bson:"family,omitempty"`
	Type   uint32  `json:"type" bson:"type,omitempty"`
	Laddr  Addr    `json:"localaddr" bson:"localaddr,omitempty"`
	Raddr  Addr    `json:"remoteaddr" bson:"remoteaddr,omitempty"`
	Status string  `json:"status" bson:"status,omitempty"`
	Uids   []int32 `json:"uids" bson:"uids,omitempty"`
	Pid    int32   `json:"pid" bson:"pid,omitempty"`
}

// System wide stats about different network protocols
type ProtoCountersStat struct {
	Protocol string           `json:"protocol" bson:"protocol,omitempty"`
	Stats    map[string]int64 `json:"stats" bson:"stats,omitempty"`
}

// NetInterfaceAddr is designed for represent interface addresses
type InterfaceAddr struct {
	Addr string `json:"addr" bson:"addr,omitempty"`
}

type InterfaceStat struct {
	MTU          int             `json:"mtu" bson:"mtu,omitempty"`                   // maximum transmission unit
	Name         string          `json:"name"`                                       // e.g., "en0", "lo0", "eth0.100" bson:"name"`         // e.g., "en0", "lo0", "eth0.100,omitempty"
	HardwareAddr string          `json:"hardwareaddr" bson:"hardwareaddr,omitempty"` // IEEE MAC-48, EUI-48 and EUI-64 form
	Flags        []string        `json:"flags" bson:"flags,omitempty"`               // e.g., FlagUp, FlagLoopback, FlagMulticast
	Addrs        []InterfaceAddr `json:"addrs" bson:"addrs,omitempty"`
}

type FilterStat struct {
	ConnTrackCount int64 `json:"conntrackCount" bson:"conntrackCount,omitempty"`
	ConnTrackMax   int64 `json:"conntrackMax" bson:"conntrackMax,omitempty"`
}

var constMap = map[string]int{
	"TCP":  syscall.SOCK_STREAM,
	"UDP":  syscall.SOCK_DGRAM,
	"IPv4": syscall.AF_INET,
	"IPv6": syscall.AF_INET6,
}

func (n IOCountersStat) String() string {
	s, _ := json.Marshal(n)
	return string(s)
}

func (n ConnectionStat) String() string {
	s, _ := json.Marshal(n)
	return string(s)
}

func (n ProtoCountersStat) String() string {
	s, _ := json.Marshal(n)
	return string(s)
}

func (a Addr) String() string {
	s, _ := json.Marshal(a)
	return string(s)
}

func (n InterfaceStat) String() string {
	s, _ := json.Marshal(n)
	return string(s)
}

func (n InterfaceAddr) String() string {
	s, _ := json.Marshal(n)
	return string(s)
}

func Interfaces() ([]InterfaceStat, error) {
	is, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	ret := make([]InterfaceStat, 0, len(is))
	for _, ifi := range is {

		var flags []string
		if ifi.Flags&net.FlagUp != 0 {
			flags = append(flags, "up")
		}
		if ifi.Flags&net.FlagBroadcast != 0 {
			flags = append(flags, "broadcast")
		}
		if ifi.Flags&net.FlagLoopback != 0 {
			flags = append(flags, "loopback")
		}
		if ifi.Flags&net.FlagPointToPoint != 0 {
			flags = append(flags, "pointtopoint")
		}
		if ifi.Flags&net.FlagMulticast != 0 {
			flags = append(flags, "multicast")
		}

		r := InterfaceStat{
			Name:         ifi.Name,
			MTU:          ifi.MTU,
			HardwareAddr: ifi.HardwareAddr.String(),
			Flags:        flags,
		}
		addrs, err := ifi.Addrs()
		if err == nil {
			r.Addrs = make([]InterfaceAddr, 0, len(addrs))
			for _, addr := range addrs {
				r.Addrs = append(r.Addrs, InterfaceAddr{
					Addr: addr.String(),
				})
			}

		}
		ret = append(ret, r)
	}

	return ret, nil
}

func getIOCountersAll(n []IOCountersStat) ([]IOCountersStat, error) {
	r := IOCountersStat{
		Name: "all",
	}
	for _, nic := range n {
		r.BytesRecv += nic.BytesRecv
		r.PacketsRecv += nic.PacketsRecv
		r.Errin += nic.Errin
		r.Dropin += nic.Dropin
		r.BytesSent += nic.BytesSent
		r.PacketsSent += nic.PacketsSent
		r.Errout += nic.Errout
		r.Dropout += nic.Dropout
	}

	return []IOCountersStat{r}, nil
}

func parseNetLine(line string) (ConnectionStat, error) {
	f := strings.Fields(line)
	if len(f) < 9 {
		return ConnectionStat{}, fmt.Errorf("wrong line,%s", line)
	}

	pid, err := strconv.Atoi(f[1])
	if err != nil {
		return ConnectionStat{}, err
	}
	fd, err := strconv.Atoi(strings.Trim(f[3], "u"))
	if err != nil {
		return ConnectionStat{}, fmt.Errorf("unknown fd, %s", f[3])
	}
	netFamily, ok := constMap[f[4]]
	if !ok {
		return ConnectionStat{}, fmt.Errorf("unknown family, %s", f[4])
	}
	netType, ok := constMap[f[7]]
	if !ok {
		return ConnectionStat{}, fmt.Errorf("unknown type, %s", f[7])
	}

	laddr, raddr, err := parseNetAddr(f[8])
	if err != nil {
		return ConnectionStat{}, fmt.Errorf("failed to parse netaddr, %s", f[8])
	}

	n := ConnectionStat{
		Fd:     uint32(fd),
		Family: uint32(netFamily),
		Type:   uint32(netType),
		Laddr:  laddr,
		Raddr:  raddr,
		Pid:    int32(pid),
	}
	if len(f) == 10 {
		n.Status = strings.Trim(f[9], "()")
	}

	return n, nil
}

func parseNetAddr(line string) (laddr Addr, raddr Addr, err error) {
	parse := func(l string) (Addr, error) {
		host, port, err := net.SplitHostPort(l)
		if err != nil {
			return Addr{}, fmt.Errorf("wrong addr, %s", l)
		}
		lport, err := strconv.Atoi(port)
		if err != nil {
			return Addr{}, err
		}
		return Addr{IP: host, Port: uint32(lport)}, nil
	}

	addrs := strings.Split(line, "->")
	if len(addrs) == 0 {
		return laddr, raddr, fmt.Errorf("wrong netaddr, %s", line)
	}
	laddr, err = parse(addrs[0])
	if len(addrs) == 2 { // remote addr exists
		raddr, err = parse(addrs[1])
		if err != nil {
			return laddr, raddr, err
		}
	}

	return laddr, raddr, err
}
