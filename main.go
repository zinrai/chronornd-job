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

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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
	serial  bool
	mu      sync.Mutex
	running bool
}

// begin reports whether a job may start now. In serial mode it returns false
// while a previous job is still running, so the caller skips this occurrence.
func (e *JobExecutor) begin() bool {
	if !e.serial {
		return true
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return false
	}
	e.running = true
	return true
}

func (e *JobExecutor) end() {
	if !e.serial {
		return
	}
	e.mu.Lock()
	e.running = false
	e.mu.Unlock()
}

func (e *JobExecutor) execute(ctx context.Context, job Job) {
	cmd := exec.CommandContext(ctx, job.command, job.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Executing command: %s %v", job.command, job.args)
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing job: %v", err)
		return
	}
	log.Println("Command executed successfully")
}

func parseFlags() Config {
	var config Config
	var showVersion bool

	flag.StringVar(&config.command, "command", "", "Command to execute")
	flag.IntVar(&config.executions, "n", 10, "Number of executions per day")
	flag.BoolVar(&config.serialExec, "serial", false, "Execute jobs serially (skip if previous job is running)")
	flag.BoolVar(&showVersion, "version", false, "Print version information and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [-- COMMAND_ARGS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("chronornd-job version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

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

	executor := &JobExecutor{serial: config.serialExec}
	jobQueue := generateJobs(r, config, time.Now())

	log.Println("Planned execution times:")
	for _, job := range jobQueue {
		log.Printf("  %s", job.execTime.Format("15:04:05"))
	}

	var wg sync.WaitGroup
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
			wg.Wait()
			return ctx.Err()
		case <-timer.C:
			if !executor.begin() {
				log.Printf("Skipping job at %s: previous job is still running",
					job.execTime.Format("15:04:05"))
				continue
			}
			wg.Add(1)
			go func(job Job) {
				defer wg.Done()
				defer executor.end()
				executor.execute(ctx, job)
			}(job)
		}
	}

	wg.Wait()
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
