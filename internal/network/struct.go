package network

import (
	"net"

	"github.com/jetkvm/kvm/internal/udhcpc"
	"github.com/rs/zerolog"
)

type NetworkInterfaceState struct {
	Options       *NetworkInterfaceOptions
	interfaceName string
	interfaceUp   bool
	ipv4Addr      *net.IP
	ipv4Addresses []string
	ipv6Addr      *net.IP
	ipv6Addresses []IPv6Address
	ipv6LinkLocal *net.IP
	ntpAddresses  []*net.IP
	macAddr       *net.HardwareAddr

	l *zerolog.Logger

	config     *NetworkConfig
	DhcpClient *udhcpc.DHCPClient

	defaultHostname string
	currentHostname string
	currentFqdn     string

	onStateChange func(state *NetworkInterfaceState)
}

type NetworkInterfaceOptions struct {
	InterfaceName     string
	DhcpPidFile       string
	Logger            *zerolog.Logger
	DefaultHostname   string
	EventQueue        chan NetworkEvent
	OnStateChange     func(state *NetworkInterfaceState)
	OnDhcpLeaseChange func(state *NetworkInterfaceState, lease *udhcpc.Lease)
	NetworkConfig     *NetworkConfig
}

type EventType int

const (
	EventTypeInformational EventType = iota
	EventTypeNetworkConfigChange
	EventTypeNetworkInterfaceStateChange
	EventTypeDHCPLeaseChange
)

var eventTypeName = map[EventType]string{
	EventTypeInformational:               "informational",
	EventTypeNetworkConfigChange:         "network config",
	EventTypeNetworkInterfaceStateChange: "network interface state",
	EventTypeDHCPLeaseChange:             "dhcp lease",
}

func (ss EventType) String() string {
	return eventTypeName[ss]
}

type NetworkEvent struct {
	EventType EventType
	State     *NetworkInterfaceState
	Lease     *udhcpc.Lease
	Config    *NetworkConfig
}
