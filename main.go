package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	command     string
	executions  int
	serialExec  bool
	commandArgs []string
}

type Job struct {
	execTime time.Time
	command  string
	args     []string
}

type JobExecutor struct {
	mu         sync.Mutex
	serialExec bool
	isLocked   bool
}

func NewJobExecutor(serialExec bool) *JobExecutor {
	return &JobExecutor{
		serialExec: serialExec,
	}
}

func (e *JobExecutor) executeJob(ctx context.Context, job Job) error {
	if e.serialExec {
		if !e.tryLock() {
			return fmt.Errorf("skipped: another job is running")
		}
		defer e.unlock()
	}

	cmd := exec.CommandContext(ctx, job.command, job.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Executing command: %s %v", job.command, job.args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %v", err)
	}
	log.Println("Command executed successfully")
	return nil
}

func (e *JobExecutor) tryLock() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.isLocked {
		return false
	}
	e.isLocked = true
	return true
}

func (e *JobExecutor) unlock() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.isLocked = false
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.command, "command", "", "Command to execute")
	flag.IntVar(&config.executions, "n", 10, "Number of executions per day")
	flag.BoolVar(&config.serialExec, "serial", false, "Execute jobs serially (skip if previous job is running)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [-- COMMAND_ARGS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if config.command == "" {
		config.command = "echo"
		config.commandArgs = []string{"Job executed at random time"}
	} else {
		config.commandArgs = flag.Args()
	}

	if config.executions < 1 {
		log.Fatal("Number of executions must be positive")
	}

	return config
}

func generateJobs(r *rand.Rand, config Config, now time.Time) []Job {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)

	timeRange := end.Sub(start).Seconds()
	jobs := make([]Job, config.executions)

	for i := 0; i < config.executions; i++ {
		randomSeconds := r.Float64() * timeRange
		execTime := start.Add(time.Duration(randomSeconds) * time.Second)
		jobs[i] = Job{
			execTime: execTime,
			command:  config.command,
			args:     config.commandArgs,
		}
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].execTime.Before(jobs[j].execTime)
	})

	return jobs
}

func run(ctx context.Context, config Config, r *rand.Rand) error {
	log.Printf("Starting chronornd-job (Command: %s, Executions: %d, Serial: %v)",
		config.command,
		config.executions,
		config.serialExec)

	executor := NewJobExecutor(config.serialExec)
	jobQueue := generateJobs(r, config, time.Now())

	log.Println("Planned execution times:")
	for _, job := range jobQueue {
		log.Printf("  %s", job.execTime.Format("15:04:05"))
	}

	for _, job := range jobQueue {
		if time.Now().After(job.execTime) {
			log.Printf("Skipping past job scheduled for %s", job.execTime.Format("15:04:05"))
			continue
		}

		waitDuration := time.Until(job.execTime)
		timer := time.NewTimer(waitDuration)

		log.Printf("Waiting for %v until next execution at %s",
			waitDuration.Round(time.Second),
			job.execTime.Format("15:04:05"))

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			if err := executor.executeJob(ctx, job); err != nil {
				if config.serialExec && err.Error() == "skipped: another job is running" {
					log.Printf("Skipping job at %s: previous job is still running",
						job.execTime.Format("15:04:05"))
				} else {
					log.Printf("Error executing job: %v", err)
				}
			}
		}
	}

	log.Println("All jobs completed. Exiting...")
	return nil
}

func main() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	config := parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	if err := run(ctx, config, r); err != nil && err != context.Canceled {
		log.Fatalf("Error: %v", err)
	}
}
