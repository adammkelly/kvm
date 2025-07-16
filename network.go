package kvm

import (
	"fmt"
	"time"

	"github.com/jetkvm/kvm/internal/network"
	"github.com/jetkvm/kvm/internal/udhcpc"
)

const (
	NetIfName = "eth0"
)

var (
	networkState *network.NetworkInterfaceState
)

func networkStateChanged() {
	networkLogger.Info().Msg("NTW CHANGE start")
	// do not block the main thread
	go waitCtrlAndRequestDisplayUpdate(true)
	networkLogger.Info().Msg("Continue ...")

	if timeSync != nil {
		if networkState != nil {
			timeSync.SetDhcpNtpAddresses(networkState.NtpAddressesString())
		}

		if err := timeSync.Sync(); err != nil {
			networkLogger.Error().Err(err).Msg("failed to sync time after network state change")
		}
	}

	// always restart mDNS when the network state changes
	if mDNS != nil {
		_ = mDNS.SetListenOptions(config.NetworkConfig.GetMDNSMode())
		_ = mDNS.SetLocalNames([]string{
			networkState.GetHostname(),
			networkState.GetFQDN(),
		}, true)
	}
	networkLogger.Info().Msg("NTW CHANGE finish")
}

func worker(queue chan network.NetworkEvent) {
	networkLogger.Info().Msg("Working start")
	for {
		event := <-queue
		networkLogger.Info().Msg("Working")
		evt_string := fmt.Sprintf(" Got event:%s", event.EventType.String())
		networkLogger.Info().Msg(evt_string)
		switch event.EventType {
		case network.EventTypeNetworkInterfaceStateChange:
			networkStateChanged()
		default:
			evt_string := fmt.Sprintf(" Got unhandled event:%s", event.EventType.String())
			networkLogger.Info().Msg(evt_string)

		}
		networkLogger.Info().Msg("Working")
		time.Sleep(time.Second)
	}
}

func interfaceModeDHCP(queue chan network.NetworkEvent) (*network.NetworkInterfaceState, error) {
	networkLogger.Info().Msg("interfaceModeDHCP start")

	state, err := network.NewNetworkInterfaceState(&network.NetworkInterfaceOptions{
		DefaultHostname: GetDefaultHostname(),
		InterfaceName:   NetIfName,
		NetworkConfig:   config.NetworkConfig,
		Logger:          networkLogger,
		EventQueue:      queue,
		OnStateChange: func(state *network.NetworkInterfaceState) {
			networkStateChanged()
			c := state.Options.EventQueue
			c <- network.NetworkEvent{EventType: network.EventTypeNetworkInterfaceStateChange, State: state}
		},
		OnInitialCheck: func(state *network.NetworkInterfaceState) {
			networkStateChanged()
			c := state.Options.EventQueue
			c <- network.NetworkEvent{EventType: network.EventTypeNetworkInterfaceStateChange, State: state}
		},
		OnDhcpLeaseChange: func(state *network.NetworkInterfaceState, lease *udhcpc.Lease) {
			c := state.Options.EventQueue
			c <- network.NetworkEvent{EventType: network.EventTypeNetworkInterfaceStateChange, State: state}
			networkStateChanged()

			if currentSession == nil {
				return
			}

			writeJSONRPCEvent("networkState", networkState.RpcGetNetworkState(), currentSession)
		},
		OnConfigChange: func(state *network.NetworkInterfaceState, networkConfig *network.NetworkConfig) {
			config.NetworkConfig = networkConfig
			networkStateChanged()
		},
	})

	if err != nil {
		return state, fmt.Errorf("failed to create NetworkInterfaceState")
	}

	if err := state.DhcpClient.Run(); err != nil {
		networkLogger.Info().Msg("interfaceModeDHCP end")
		return state, err
	}
	networkLogger.Info().Msg("interfaceModeDHCP end")

	return state, nil
}

func interfaceModeStaticIP(queue chan network.NetworkEvent) (*network.NetworkInterfaceState, error) {

	state, err := network.NewNetworkInterfaceState(&network.NetworkInterfaceOptions{
		DefaultHostname: GetDefaultHostname(),
		InterfaceName:   NetIfName,
		NetworkConfig:   config.NetworkConfig,
		Logger:          networkLogger,
		EventQueue:      queue,
	})

	return state, err
}

func initNetwork() error {
	ensureConfigLoaded()
	// Ensure we have choosen dhcp or static
	mode := config.NetworkConfig.IPv4Mode.String
	sss := fmt.Sprintf("AMKKKKKKKKKKKKKKKKKK - %s", mode)
	networkLogger.Info().Msg(sss)
	cc := make(chan network.NetworkEvent)

	var err error
	var state *network.NetworkInterfaceState

	go worker(cc)

	switch mode {
	case "dhcp":
		state, err = interfaceModeDHCP(cc)
		networkLogger.Error().Err(err).Msg("failed to setup network mode")
	case "static":
		state, err = interfaceModeStaticIP(cc)
	default:
		networkLogger.Error().Msg("Unknown IP mode selected.")
	}
	if err != nil {
		networkLogger.Error().Err(err).Msg("failed to setup network mode")
		return err

	}

	// We assume only ever one network at a time, this might be fine for now but potentially could be many
	// One of ipv4, one for ipv6. Also could have several networks too.
	networkState = state

	return nil
}

func rpcGetNetworkState() network.RpcNetworkState {
	return networkState.RpcGetNetworkState()
}

func rpcGetNetworkSettings() network.RpcNetworkSettings {
	return networkState.RpcGetNetworkSettings()
}

func rpcSetNetworkSettings(settings network.RpcNetworkSettings) (*network.RpcNetworkSettings, error) {
	s := networkState.RpcSetNetworkSettings(settings)
	if s != nil {
		return nil, s
	}

	if err := SaveConfig(); err != nil {
		return nil, err
	}

	return &network.RpcNetworkSettings{NetworkConfig: *config.NetworkConfig}, nil
}

func rpcRenewDHCPLease() error {
	return networkState.RpcRenewDHCPLease()
}
