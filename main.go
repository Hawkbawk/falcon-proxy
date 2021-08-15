package main

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const DefaultContainerName = "falcon-proxy"

func main() {
	syncer, err := NewProxySyncer()

	if err != nil {
		log.Fatalln("Unable to create a Docker client. This is likely due to a bug. ERROR:", err)
	}

	syncer.Start()
}


type ProxySyncer struct {
	Client *client.Client
	Context context.Context
	ContainerID string
	EventChannel <-chan events.Message
}

type Event struct {
	Message events.Message
	Finished bool
}

// Creates a new Syncer struct for syncing a container with all Docker networks on a machine.
// Returns an empty struct and an error if it was unable to construct the Docker client.
func NewProxySyncer() (ProxySyncer, error) {
	// The most up-to-date client version as of writing this code. Ensures object shapes and other
	// such stuff doesn't change on us depending on the user's version of Docker
	client, err := client.NewClientWithOpts(client.WithVersion("20.10.7"), client.FromEnv)
	if err != nil {
		return ProxySyncer{}, nil
	}
	context := context.Background()
	eventChannel, _ := client.Events(context, types.EventsOptions{Filters: createFilters()})


	return ProxySyncer{
		Client: client,
		Context: context,
		ContainerID: DefaultContainerName,
		EventChannel: eventChannel,
	}, nil
}

// StartSyncing listens to the Docker API for any network changes and either adds or removes the
// falcon-proxy as necessary, whenever any changes come in.
func (syncer ProxySyncer) Start() {
	syncer.listenForNetworkEvents()
}

// listenForNetworkEvents reads in Docker events from the passed in channel and either adds the proxy
// to the network, removes the proxy from the network, or does nothing, depending on what's necessary.
// Note that this method is blocking and will never return, as it is designed to run for perpetuity.
func (syncer ProxySyncer) listenForNetworkEvents() {
	for {
		event := <- syncer.EventChannel
		connectedNetworks, err := syncer.connectedNetworks()

		if err != nil {
			log.Println("Unable to read what networks the proxy is connected to. ERROR:", err)
			continue
		} else if connectedNetworks[event.Actor.ID] == nil {
			continue
		}

		switch event.Action {
			case "destroy": {
				syncer.leaveNetwork(event.Actor.ID)
				break
			}
			case "create": {
				syncer.joinNetwork(event.Actor.ID)
				break
			}
		}
	}
}

// joinNetwork adds the proxy container to the specified network.
func (s ProxySyncer) joinNetwork(changedNetworkID string) error {
	if err := s.Client.NetworkDisconnect(s.Context, changedNetworkID, s.ContainerID, true); err != nil {
		return err
	}
	return nil
}

// leaveNetwork removes the proxy container from the specified network.
func (s ProxySyncer) leaveNetwork(changedNetworkID string) error {
	err := s.Client.NetworkConnect(s.Context, changedNetworkID, s.ContainerID, &network.EndpointSettings{})
	if err != nil {
		return err
	}
	return nil
}

// connectedNetworks returns what networks the proxy container is already a part of.
func (s ProxySyncer) connectedNetworks() (map[string]*(network.EndpointSettings), error) {
	container, err := s.Client.ContainerInspect(context.Background(), s.ContainerID)

	if err != nil {
		return nil, err
	}

	return container.NetworkSettings.Networks, nil
}

// createFilters filters the events from Docker to only relate to network creation and deletion
// events.
func createFilters() filters.Args {
	args := filters.NewArgs()

	args.Add("type", "network")
	args.Add("event", "create")
	args.Add("event", "destroy")

	return args
}







