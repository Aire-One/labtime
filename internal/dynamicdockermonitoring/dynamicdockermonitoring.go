package dynamicdockermonitoring

import (
	"context"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

type DynamicDockerMonitor struct {
	client *client.Client

	Events chan events.Message
	Errors chan error
}

func NewDynamicDockerMonitor(ctx context.Context) (*DynamicDockerMonitor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	eventchan := make(chan events.Message, 1)
	errchan := make(chan error, 1)
	go watch(ctx, cli, eventchan, errchan)

	return &DynamicDockerMonitor{
		client: cli,
		Events: eventchan,
		Errors: errchan,
	}, nil
}

func (d *DynamicDockerMonitor) Shutdown() error {
	return d.client.Close()
}

func watch(ctx context.Context, cli *client.Client, eventchan chan events.Message, errchan chan error) {
	eventStream, errs := cli.Events(ctx, events.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("type", "container"),
			filters.Arg("event", "create"),
		),
	})

	for {
		select {
		case event := <-eventStream:
			eventchan <- event
		case err := <-errs:
			if err != nil && !errors.Is(err, context.Canceled) {
				errchan <- errors.Wrap(err, "error receiving docker event for dynamic monitoring")
			}
			return
		}
	}
}
