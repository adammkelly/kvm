package network

import (
	"fmt"
	"net"

	"github.com/jetkvm/kvm/internal/confparser"
	"github.com/jetkvm/kvm/internal/logging"
	"github.com/jetkvm/kvm/internal/udhcpc"

	"github.com/vishvananda/netlink"
)

var GLOB_IF_LIST [1]*NetworkInterfaceState

func findNetworkInterface(if_name string) *NetworkInterfaceState {
	for _, if_obj := range GLOB_IF_LIST {
		if if_obj.interfaceName == if_name {
			return if_obj
		}
	}
	return nil
}

func parseIPMask(mask string) (net.IPMask, error) {
	ip := net.ParseIP(mask)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address format for mask: %s", mask)
	}
	ipv4 := ip.To4()
	if ipv4 == nil {
		return nil, fmt.Errorf("mask %s is not an IPv4 address", mask)
	}
	return net.IPMask(ipv4), nil
}

func NewNetworkInterfaceState(opts *NetworkInterfaceOptions) (*NetworkInterfaceState, error) {
	if opts.NetworkConfig == nil {
		return nil, fmt.Errorf("NetworkConfig can not be nil")
	}

	if opts.DefaultHostname == "" {
		opts.DefaultHostname = "jetkvm"
	}

	err := confparser.SetDefaultsAndValidate(opts.NetworkConfig)
	if err != nil {
		return nil, err
	}

	l := opts.Logger
	s := &NetworkInterfaceState{
		interfaceName:   opts.InterfaceName,
		defaultHostname: opts.DefaultHostname,
		l:               l,
		onStateChange:   opts.OnStateChange,
		config:          opts.NetworkConfig,
		ntpAddresses:    make([]*net.IP, 0),
		Options:         opts,
	}
	GLOB_IF_LIST[0] = s

	mode := opts.NetworkConfig.IPv4Mode.String

	// create the dhcp client

	if mode == "dhcp" {
		s.l.Info().Msg("DHCP UP")
		dhcpClient := udhcpc.NewDHCPClient(&udhcpc.DHCPClientOptions{
			InterfaceName: opts.InterfaceName,
			PidFile:       opts.DhcpPidFile,
			Logger:        l,
			OnLeaseChange: func(lease *udhcpc.Lease) {
				s.update()
				//if err != nil {
				//	opts.Logger.Error().Err(err).Msg("failed to update network state")
				//	return
				//}
				s.updateNtpServersFromLease(lease)
				s.setHostnameIfNotSame()
				interf := findNetworkInterface(lease.Client.InterfaceName)
				opts.OnDhcpLeaseChange(interf, lease)

			},
		})
		s.DhcpClient = dhcpClient

		s.l.Info().Msg("DHCP UP2")
	} else {
		static_info := opts.NetworkConfig.IPv4Static
		s.l.Info().Msg("STATIC TIME?!")

		ipMask, err := parseIPMask(static_info.Netmask.String)
		if err != nil {
			fmt.Printf("Failed to parse IP mask: %v\n", err)
			return s, err
		}

		pl, _ := ipMask.Size()
		fmt.Printf("Subnet mask is a /%d\n", pl)
		eth0, _ := netlink.LinkByName(opts.InterfaceName)
		// netlink.ParseAddr()
		ipAddress := fmt.Sprintf("%s/%d", static_info.Address.String, pl)
		addr, _ := netlink.ParseAddr(ipAddress)
		netlink.AddrAdd(eth0, addr)
	}

	return s, nil
}

func (s *NetworkInterfaceState) Close() {

	mode := s.config.IPv4Mode.String

	// create the dhcp client

	if mode == "dhcp" {
		s.DhcpClient = nil
	}
	GLOB_IF_LIST[0] = nil
}

func (s *NetworkInterfaceState) IsUp() bool {
	return s.interfaceUp
}

func (s *NetworkInterfaceState) HasIPAssigned() bool {
	return s.ipv4Addr != nil || s.ipv6Addr != nil
}

func (s *NetworkInterfaceState) IsOnline() bool {
	return s.IsUp() && s.HasIPAssigned()
}

func (s *NetworkInterfaceState) IPv4() *net.IP {
	return s.ipv4Addr
}

func (s *NetworkInterfaceState) IPv4String() string {
	if s.ipv4Addr == nil {
		return "..."
	}
	return s.ipv4Addr.String()
}

func (s *NetworkInterfaceState) IPv6() *net.IP {
	return s.ipv6Addr
}

func (s *NetworkInterfaceState) IPv6String() string {
	if s.ipv6Addr == nil {
		return "..."
	}
	return s.ipv6Addr.String()
}

func (s *NetworkInterfaceState) NtpAddresses() []*net.IP {
	return s.ntpAddresses
}

func (s *NetworkInterfaceState) NtpAddressesString() []string {
	ntpServers := []string{}

	if s != nil {
		s.l.Debug().Any("s", s).Msg("getting NTP address strings")

		if len(s.ntpAddresses) > 0 {
			for _, server := range s.ntpAddresses {
				s.l.Debug().IPAddr("server", *server).Msg("converting NTP address")
				ntpServers = append(ntpServers, server.String())
			}
		}
	}

	return ntpServers
}

func (s *NetworkInterfaceState) MAC() *net.HardwareAddr {
	return s.macAddr
}

func (s *NetworkInterfaceState) MACString() string {
	if s.macAddr == nil {
		return ""
	}
	return s.macAddr.String()
}

func (s *NetworkInterfaceState) hasInterfaceStateChanged(iface netlink.Link) (bool, bool) {
	// detect if the interface status changed
	var changed bool
	attrs := iface.Attrs()
	state := attrs.OperState
	newInterfaceUp := state == netlink.OperUp

	// check if the interface is coming up
	interfaceGoingUp := !s.interfaceUp && newInterfaceUp

	if s.interfaceUp != newInterfaceUp {
		s.interfaceUp = newInterfaceUp
		changed = true
	}

	return changed, interfaceGoingUp
}

func (s *NetworkInterfaceState) hasIpv4AddressChanged(ipv4Addresses []net.IP) bool {
	// detect if the interface status changed
	var changed bool
	if len(ipv4Addresses) > 0 {
		// compare the addresses to see if there's a change
		if s.ipv4Addr == nil || s.ipv4Addr.String() != ipv4Addresses[0].String() {
			scopedLogger := s.l.With().Str("ipv4", ipv4Addresses[0].String()).Logger()
			if s.ipv4Addr != nil {
				scopedLogger.Info().
					Str("old_ipv4", s.ipv4Addr.String()).
					Msg("IPv4 address changed")
			} else {
				scopedLogger.Info().Msg("IPv4 address found")
			}
			s.ipv4Addr = &ipv4Addresses[0]
			changed = true
		}
	}

	return changed
}

func (s *NetworkInterfaceState) hasIpv6LinkLocalChanged(ipv6LinkLocal *net.IP) bool {
	// detect if the interface status changed
	var changed bool
	if s.ipv6LinkLocal == nil || s.ipv6LinkLocal.String() != ipv6LinkLocal.String() {
		scopedLogger := s.l.With().Str("ipv6", ipv6LinkLocal.String()).Logger()
		if s.ipv6LinkLocal != nil {
			scopedLogger.Info().
				Str("old_ipv6", s.ipv6LinkLocal.String()).
				Msg("IPv6 link local address changed")
		} else {
			scopedLogger.Info().Msg("IPv6 link local address found")
		}
		s.ipv6LinkLocal = ipv6LinkLocal
		changed = true
	}

	return changed
}

func (s *NetworkInterfaceState) hasIpv6AddressChanged(ipv6Addresses []IPv6Address) bool {
	// detect if the interface status changed
	var changed bool
	if len(ipv6Addresses) > 0 {
		// compare the addresses to see if there's a change
		if s.ipv6Addr == nil || s.ipv6Addr.String() != ipv6Addresses[0].Address.String() {
			scopedLogger := s.l.With().Str("ipv6", ipv6Addresses[0].Address.String()).Logger()
			if s.ipv6Addr != nil {
				scopedLogger.Info().
					Str("old_ipv6", s.ipv6Addr.String()).
					Msg("IPv6 address changed")
			} else {
				scopedLogger.Info().Msg("IPv6 address found")
			}
			s.ipv6Addr = &ipv6Addresses[0].Address
			changed = true
		}
	}

	return changed
}

func (s *NetworkInterfaceState) update() (DhcpTargetState, error) {
	// Remove this lock by having the worker take care of all updates.

	dhcpTargetState := DhcpTargetStateDoNothing

	iface, err := netlink.LinkByName(s.interfaceName)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get interface")
		return dhcpTargetState, err
	}

	attrs := iface.Attrs()

	// IF change
	changed, interfaceGoingUp := s.hasInterfaceStateChanged(iface)

	if changed {
		if interfaceGoingUp {
			s.l.Info().Msg("interface state transitioned to up")
			dhcpTargetState = DhcpTargetStateRenew
		} else {
			s.l.Info().Msg("interface state transitioned to down")
		}
	}
	// set the mac address
	s.macAddr = &attrs.HardwareAddr

	// get the ip addresses
	// Gets both from ifconfig
	addrs, err := netlinkAddrs(iface)
	if err != nil {
		return dhcpTargetState, logging.ErrorfL(s.l, "failed to get ip addresses", err)
	}

	var (
		ipv4Addresses       = make([]net.IP, 0)
		ipv4AddressesString = make([]string, 0)
		ipv6Addresses       = make([]IPv6Address, 0)
		// ipv6AddressesString = make([]string, 0)
		ipv6LinkLocal *net.IP
	)

	for _, addr := range addrs {
		if addr.IP.To4() != nil {
			scopedLogger := s.l.With().Str("ipv4", addr.IP.String()).Logger()
			if !interfaceGoingUp {
				// remove all IPv4 addresses from the interface.
				scopedLogger.Info().Msg("state transitioned to down, removing IPv4 address")
				err := netlink.AddrDel(iface, &addr)
				if err != nil {
					scopedLogger.Warn().Err(err).Msg("failed to delete address")
				}
				// notify the DHCP client to release the lease
				dhcpTargetState = DhcpTargetStateRelease
				continue
			}
			ipv4Addresses = append(ipv4Addresses, addr.IP)
			ipv4AddressesString = append(ipv4AddressesString, addr.IPNet.String())
		} else if addr.IP.To16() != nil {
			scopedLogger := s.l.With().Str("ipv6", addr.IP.String()).Logger()
			// check if it's a link local address
			if addr.IP.IsLinkLocalUnicast() {
				ipv6LinkLocal = &addr.IP
				continue
			}

			if !addr.IP.IsGlobalUnicast() {
				scopedLogger.Trace().Msg("not a global unicast address, skipping")
				continue
			}

			if !interfaceGoingUp {
				scopedLogger.Info().Msg("state transitioned to down, removing IPv6 address")
				err := netlink.AddrDel(iface, &addr)
				if err != nil {
					scopedLogger.Warn().Err(err).Msg("failed to delete address")
				}
				continue
			}
			ipv6Addresses = append(ipv6Addresses, IPv6Address{
				Address:           addr.IP,
				Prefix:            *addr.IPNet,
				ValidLifetime:     lifetimeToTime(addr.ValidLft),
				PreferredLifetime: lifetimeToTime(addr.PreferedLft),
				Scope:             addr.Scope,
			})
			// ipv6AddressesString = append(ipv6AddressesString, addr.IPNet.String())
		}
	}

	change_happened := s.hasIpv4AddressChanged(ipv4Addresses)
	if change_happened {
		changed = true
	}
	s.ipv4Addresses = ipv4AddressesString

	if ipv6LinkLocal != nil {
		change_happened = s.hasIpv6LinkLocalChanged(ipv6LinkLocal)
	}
	s.ipv6Addresses = ipv6Addresses

	if change_happened {
		changed = true
	}

	change_happened = s.hasIpv6AddressChanged(ipv6Addresses)

	if change_happened {
		changed = true
	}

	if changed {
		s.onStateChange(s)
	}
	s.l.Info().Msg(fmt.Sprintf("Default format: %v\n", s))

	return dhcpTargetState, nil
}

func (s *NetworkInterfaceState) updateNtpServersFromLease(lease *udhcpc.Lease) error {
	if lease != nil && len(lease.NTPServers) > 0 {
		s.l.Info().Msg("lease found, updating DHCP NTP addresses")
		s.ntpAddresses = make([]*net.IP, 0, len(lease.NTPServers))

		for _, ntpServer := range lease.NTPServers {
			if ntpServer != nil {
				s.l.Info().IPAddr("ntp_server", ntpServer).Msg("NTP server found in lease")
				s.ntpAddresses = append(s.ntpAddresses, &ntpServer)
			}
		}
	} else {
		s.l.Info().Msg("no NTP servers found in lease")
		s.ntpAddresses = make([]*net.IP, 0, len(s.config.TimeSyncNTPServers))
	}

	return nil
}

func (s *NetworkInterfaceState) CheckAndUpdateDhcp() error {
	dhcpTargetState, err := s.update()
	if err != nil {
		return logging.ErrorfL(s.l, "failed to update network state", err)
	}

	switch dhcpTargetState {
	case DhcpTargetStateRenew:
		s.l.Info().Msg("renewing DHCP lease")
		_ = s.DhcpClient.Renew()
	case DhcpTargetStateRelease:
		s.l.Info().Msg("releasing DHCP lease")
		_ = s.DhcpClient.Release()
	case DhcpTargetStateStart:
		s.l.Warn().Msg("dhcpTargetStateStart not implemented")
	case DhcpTargetStateStop:
		s.l.Warn().Msg("dhcpTargetStateStop not implemented")
	}

	return nil
}
