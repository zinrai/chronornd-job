# chronornd-job

A command-line tool that executes specified commands at random times throughout the day, using a time-based queue implementation.

## Overview

`chronrnd-job` schedules and executes commands at random times during a 24-hour period. It maintains a sorted queue of jobs and executes them sequentially at their scheduled times.

This makes it useful for:

- Random testing and monitoring
- Distributed task execution
- Load testing

Use with cron for daily scheduling:

```bash
# Schedule 5 random executions daily
0 0 * * * /usr/local/bin/chronornd-job -command="/path/to/script.sh" -n=5
```

```bash
# Schedule 5 random executions daily with serial execution
0 0 * * * /usr/local/bin/chronrnd-job -command="/path/to/script.sh" -n=5 -serial
```

## Installation

Build from source:

```bash
$ go build
```

## Usage

Basic command format:

```bash
$ chronrnd-job [OPTIONS] [-- COMMAND_ARGS]
```

### Options

| Option     | Description                                             | Default                              |
|------------|---------------------------------------------------------|--------------------------------------|
| `-command` | Command to execute                                      | `echo "Job executed at random time"` |
| `-n`       | Number of executions per day                            | `10`                                 |
| `-serial`  | Execute jobs serially (skip if previous job is running) | `false`                              |

### Examples

Run a backup script 5 times throughout the day:

```bash
$ chronrnd-job -command="./backup.sh" -n=5
```

Execute rsync with arguments 3 times:

```bash
$ chronrnd-job -command="rsync" -n=3 -- -av /src /dst
```

Execute shell commands with pipes:

```bash
$ chronrnd-job -command="sh" -n=2 -- -c "date | tee /tmp/timestamp.log"
```

Run in serial mode (skip jobs if previous is still running):

```bash
$ chronrnd-job -command="./long-running-task.sh" -n=5 -serial
```

## Design

### Features

- **Random Scheduling**: Jobs are scheduled at random times throughout the day
- **Sequential Execution**: Jobs are executed in chronological order
- **Serial Execution Mode**: Optional mode to skip jobs if previous job is still running
- **Signal Handling**: Graceful shutdown on SIGINT and SIGTERM
- **Command Arguments**: Full support for command-line arguments and pipes

### Job Queue Implementation

The program uses a simple queue-based design:

1. Generates the specified number of random execution times
2. Sorts them chronologically
3. Executes jobs sequentially from the queue
4. In serial mode, skips jobs if previous job is still running

### Logging

The program provides detailed logging of:

- Startup configuration
- Planned execution times
- Job execution status
- Error conditions
- Skipped jobs (when running in serial mode)

Example output (normal mode):

```
2024/11/25 10:00:00 Starting chronornd-job (Command: ./backup.sh, Executions: 5, Serial: false)
2024/11/25 10:00:00 Planned execution times:
2024/11/25 10:00:00   11:23:45
2024/11/25 10:00:00   14:15:30
2024/11/25 10:00:00   16:48:12
```

Example output (serial mode):

```
2024/11/25 10:00:00 Starting chronornd-job (Command: ./long-job.sh, Executions: 3, Serial: true)
2024/11/25 10:00:00 Planned execution times:
2024/11/25 10:00:00   11:00:00
2024/11/25 10:00:00   11:05:00
2024/11/25 10:00:00   11:10:00
2024/11/25 11:00:00 Executing command: ./long-job.sh
2024/11/25 11:05:00 Skipping job at 11:05:00: previous job is still running
2024/11/25 11:10:00 Skipping job at 11:10:00: previous job is still running
```

## License

This project is licensed under the [MIT License](./LICENSE).
