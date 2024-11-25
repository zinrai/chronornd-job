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
	"syscall"
	"time"
)

type Config struct {
	command     string
	executions  int
	daemonMode  bool
	commandArgs []string
}

type Job struct {
	execTime time.Time
	command  string
	args     []string
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.command, "command", "", "Command to execute")
	flag.IntVar(&config.executions, "n", 10, "Number of executions per day")
	flag.BoolVar(&config.daemonMode, "daemon", false, "Run in daemon mode")

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

func generateJobs(r *rand.Rand, config Config, startTime time.Time) []Job {
	endTime := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 23, 59, 59, 0, startTime.Location())
	if startTime.After(endTime) {
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day()+1, 0, 0, 0, 0, startTime.Location())
		endTime = endTime.Add(24 * time.Hour)
	}

	timeRange := endTime.Sub(startTime).Seconds()
	jobs := make([]Job, config.executions)

	for i := 0; i < config.executions; i++ {
		randomSeconds := r.Float64() * timeRange
		execTime := startTime.Add(time.Duration(randomSeconds) * time.Second)
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

func executeJob(ctx context.Context, job Job) error {
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

func runDaemon(ctx context.Context, config Config, r *rand.Rand) error {
	log.Printf("Starting chronornd-job (Command: %s, Executions: %d, Mode: %s)",
		config.command,
		config.executions,
		map[bool]string{true: "daemon", false: "one-time"}[config.daemonMode])

	jobQueue := generateJobs(r, config, time.Now())
	log.Println("Planned execution times:")
	for _, job := range jobQueue {
		log.Printf("  %s", job.execTime.Format("15:04:05"))
	}

	for {
		if len(jobQueue) == 0 {
			if !config.daemonMode {
				log.Println("Daily execution completed. Exiting...")
				return nil
			}
			jobQueue = generateJobs(r, config, time.Now())
			log.Println("Generated new execution times for next period:")
			for _, job := range jobQueue {
				log.Printf("  %s", job.execTime.Format("15:04:05"))
			}
		}

		job := jobQueue[0]
		jobQueue = jobQueue[1:]

		waitDuration := time.Until(job.execTime)
		if waitDuration > 0 {
			log.Printf("Waiting for %v until next execution at %s",
				waitDuration.Round(time.Second),
				job.execTime.Format("15:04:05"))

			timer := time.NewTimer(waitDuration)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				if err := executeJob(ctx, job); err != nil {
					log.Printf("Error: %v", err)
				}
			}
		}
	}
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

	if err := runDaemon(ctx, config, r); err != nil && err != context.Canceled {
		log.Fatalf("Daemon error: %v", err)
	}
}
