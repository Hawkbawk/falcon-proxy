package syncer

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const ProxyContainerName = "falcon-proxy"

// Syncer is the class responsible for ensuring that the Traefik container stays in sync with
// all Docker networks on the machine, joining them when they're created and leaving them when
// they're destroyed. Note that an empty Syncer will NOT work. You must call the `NewSyncer`
// function instead.
type Syncer struct {
	Client       *client.Client
	Context      context.Context
	CancelFunc   context.CancelFunc
	EventChannel <-chan events.Message
	ContainerId  string
}

// Creates a new ProxySyncer struct for syncing a container with all Docker networks on a machine.
// Returns an empty struct and an error if it was unable to construct the Docker client.
func NewSyncer() (Syncer, error) {
	client, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.FromEnv)
	if err != nil {
		return Syncer{}, fmt.Errorf("unable to communicate with the Docker client. You might have forgotten to bind-mount the docker socket. ERROR: %v", err)
	}
	context, cancelFunc := context.WithCancel(context.Background())
	eventChannel, _ := client.Events(context, types.EventsOptions{Filters: createFilters()})

	containers, err := client.ContainerList(context, types.ContainerListOptions{All: true, Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: ProxyContainerName})})

	if err != nil {
		cancelFunc()
		return Syncer{}, fmt.Errorf("unable to inspect proxy container. ERROR: %v", err)
	} else if len(containers) != 1 {
		cancelFunc()
		return Syncer{}, fmt.Errorf("there isn't a single proxy container to inspect")
	}

	return Syncer{
		Client:       client,
		Context:      context,
		CancelFunc:   cancelFunc,
		EventChannel: eventChannel,
		ContainerId:  containers[0].ID,
	}, nil
}

// Sync determines what networks the proxy container needs to join and which networks it needs to
// leave and joins and leaves those networks as appropriate. It does this by determining which
// networks the syncer container is already a part of and which networks are considered valid. If
// the syncer hasn't joined a valid network, it joins it. If the syncer is still part of an invalid
// network, then it leaves it.
//
// While this is more expensive to compute than say, listening to all
// network connect/leave events and determining the action based on the network event, it is
// much more durable. If something goes wrong and one sync errors for some strange reason or if we
// miss an event, we can "catch up" on the next network event, rather than be stuck in an invalid
// state until the container is restarted.
func (syncer Syncer) Sync() error {
	validNetworks, err := syncer.validNetworks()
	if err != nil {
		return err
	}

	log.Println("List of valid networks:", validNetworks)
	connectedNetworks, err := syncer.connectedNetworks()
	if err != nil {
		return err
	}

	for _, network := range syncer.networksToJoin(validNetworks, connectedNetworks) {
		log.Println("Attempting to join network with id: ", network)
		if err := syncer.joinNetwork(network); err != nil {
			return err
		}
	}

	for _, network := range syncer.networksToLeave(validNetworks, connectedNetworks) {
		log.Println("Attempting to leave network with id: ", network)
		if err := syncer.leaveNetwork(network); err != nil {
			return err
		}
	}

	return nil
}

// validNetworks returns a map of network IDs to booleans. If a network ID is in the map, it is
// considered a valid network that the proxy should be a part of. This method, along with
// networksToJoin and networksToLeave, is graciously taken from
// https://github.com/codekitchen/dinghy-http-proxy/blob/master/join-networks.go.
// The code there, at the time of writing, was licensed under the MIT License. The license can be
// found at https://github.com/codekitchen/dinghy-http-proxy/blob/master/LICENSE
func (syncer Syncer) validNetworks() (map[string]bool, error) {
	allNetworks, err := syncer.Client.NetworkList(syncer.Context, types.NetworkListOptions{})

	if err != nil {
		return nil, nil
	}

	validNetworks := make(map[string]bool, len(allNetworks))

	for _, n := range allNetworks {
		// For some reason, the docker daemon doesn't actually send a list of the
		// containers when you just do a network inspect, so we have to reinspect
		// every network to actually get an accurate depiction of it.
		network, err := syncer.Client.NetworkInspect(syncer.Context, n.ID, types.NetworkInspectOptions{})
		if err != nil {
			return nil, err
		}

		if syncer.isValidNetwork(network) {
			validNetworks[network.ID] = true
		}
	}

	return validNetworks, nil
}

// isValidNetwork determines if the specified network is a valid network that the proxy
// container should be a part of.
func (syncer Syncer) isValidNetwork(network types.NetworkResource) bool {
	if network.Driver == "bridge" {
		numContainers := len(network.Containers)
		_, joined := network.Containers[syncer.ContainerId]
		return network.Options["com.docker.network.bridge.default_bridge"] == "true" ||
			numContainers > 1 ||
			(numContainers == 1 && !joined)
	}
	return false
}

// networksToJoin uses the passed in information about the current network state and determines
// which networks the proxy container should join.
func (syncer Syncer) networksToJoin(validNetworks map[string]bool, connectedNetworks map[string]bool) []string {

	toJoin := make([]string, 0, len(validNetworks))

	for networkID := range validNetworks {
		if joined := connectedNetworks[networkID]; !joined {
			toJoin = append(toJoin, networkID)
		}
	}
	return toJoin
}

// networksToLeave uses the passed in information about the current network state and determines
// which networks the proxy container should join.
func (syncer Syncer) networksToLeave(validNetworks map[string]bool, connectedNetworks map[string]bool) []string {

	toLeave := make([]string, 0, len(connectedNetworks))

	for networkID := range connectedNetworks {
		if valid := validNetworks[networkID]; !valid {
			toLeave = append(toLeave, networkID)
		}
	}

	return toLeave
}

// joinNetwork adds the proxy container to the specified network.
func (s Syncer) joinNetwork(changedNetworkID string) error {
	if err := s.Client.NetworkConnect(s.Context, changedNetworkID, ProxyContainerName, &network.EndpointSettings{}); err != nil {
		return err
	}
	return nil
}

// leaveNetwork removes the proxy container from the specified network.
func (s Syncer) leaveNetwork(changedNetworkID string) error {
	err := s.Client.NetworkDisconnect(s.Context, changedNetworkID, ProxyContainerName, true)
	if err != nil {
		return err
	}
	return nil
}

// connectedNetworks returns what networks the proxy container is already a part of.
func (s Syncer) connectedNetworks() (map[string]bool, error) {
	container, err := s.Client.ContainerInspect(s.Context, ProxyContainerName)

	if err != nil {
		return nil, err
	}

	connectedNetworks := make(map[string]bool)

	for _, network := range container.NetworkSettings.Networks {
		connectedNetworks[network.NetworkID] = true
	}
	return connectedNetworks, nil
}

// createFilters filters the events from Docker to only relate to network connect and disconnect
// events.
func createFilters() filters.Args {
	args := filters.NewArgs()

	args.Add("type", "network")
	args.Add("event", "connect")
	args.Add("event", "disconnect")

	return args
}
