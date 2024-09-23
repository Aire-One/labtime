package scheduler

import (
	"log"
	"time"

	"aireone.xyz/labtime/internal/monitors"
	"github.com/go-co-op/gocron/v2"
	"github.com/pkg/errors"
)

type Scheduler struct {
	scheduler gocron.Scheduler

	logger *log.Logger
}

func NewScheduler(logger *log.Logger) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, errors.Wrap(err, "error creating scheduler")
	}

	return &Scheduler{
		scheduler: s,
		logger:    logger,
	}, nil
}

func (s *Scheduler) AddJob(m monitors.Monitor, interval int) error {
	j, err := s.scheduler.NewJob(
		gocron.DurationJob(time.Duration(interval)*time.Second),
		gocron.NewTask(func() {
			s.logger.Printf("Running job for HTTP monitor %s\n", m.ID())
			if err := m.Run(); err != nil {
				s.logger.Printf("Error running job for HTTP monitor %s: %s\n", m.ID(), err)
			}
			s.logger.Printf("Job finished for HTTP monitor %s\n", m.ID())
		}),
	)
	if err != nil {
		return errors.Wrap(err, "error creating job")
	}

	s.logger.Printf("Job started for HTTP monitor %s with ID: %s\n", m.ID(), j.ID().String())

	return nil
}

func (s *Scheduler) Start() {
	s.scheduler.Start()
}

func (s *Scheduler) Shutdown() error {
	if err := s.scheduler.Shutdown(); err != nil {
		return errors.Wrap(err, "error shutting down scheduler")
	}

	return nil
}
