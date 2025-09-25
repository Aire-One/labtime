package scheduler

import (
	"context"
	"log"
	"time"

	"aireone.xyz/labtime/internal/monitors"
	"github.com/go-co-op/gocron/v2"
	"github.com/pkg/errors"
)

const (
	FileJobTag          = "file_job"
	DynamicDockerJobTag = "dynamic_docker_job"
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

	// Start immediately the scheduler, new jobs can be added later with a start
	// immediately option
	s.Start()

	return &Scheduler{
		scheduler: s,
		logger:    logger,
	}, nil
}

func (s *Scheduler) AddJob(job monitors.Job, interval int, tag string) error {
	cronJob, err := s.scheduler.NewJob(
		gocron.DurationJob(time.Duration(interval)*time.Second),
		gocron.NewTask(func(ctx context.Context) {
			s.logger.Printf("Running job for monitor %s\n", job.ID())
			if err := job.Run(ctx); err != nil {
				s.logger.Printf("Error running job for monitor %s: %s\n", job.ID(), err)
			}
			s.logger.Printf("Job finished for monitor %s\n", job.ID())
		}),
		gocron.JobOption(gocron.WithStartImmediately()),
		gocron.WithTags(tag),
	)
	if err != nil {
		return errors.Wrap(err, "error creating job")
	}

	s.logger.Printf("Job started for monitor %s with ID: %s\n", job.ID(), cronJob.ID().String())

	return nil
}

func (s *Scheduler) RemoveByTag(tag string) {
	s.scheduler.RemoveByTags(tag)
}

func (s *Scheduler) Shutdown() error {
	if err := s.scheduler.Shutdown(); err != nil {
		return errors.Wrap(err, "error shutting down scheduler")
	}

	return nil
}
