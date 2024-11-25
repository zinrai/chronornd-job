# chronornd-job

A daemon that executes specified commands at random times throughout the day, using a time-based queue implementation.

## Overview

`chronornd-job` schedules and executes commands at random times during the day. It maintains a sorted queue of jobs and executes them sequentially at their scheduled times.

This makes it useful for:

- Random testing and monitoring
- Distributed task execution
- Load testing

## Installation

build from source:

```bash
$ go build
```

## Usage

Basic command format:

```bash
$ chronornd-job [OPTIONS] [-- COMMAND_ARGS]
```

### Options

| Option     | Description                  | Default                              |
|------------|------------------------------|--------------------------------------|
| `-command` | Command to execute           | `echo "Job executed at random time"` |
| `-n`       | Number of executions per day | `10`                                 |
| `-daemon`  | Run in daemon mode           | `false`                              |

### Examples

Run a backup script 5 times per day:

```bash
$ chronornd-job -command="./backup.sh" -n=5
```

Execute rsync with arguments 3 times per day:

```bash
$ chronornd-job -command="rsync" -n=3 -- -av /src /dst
```

Run monitoring in daemon mode:

```bash
$ chronornd-job -command="./monitor.sh" -daemon
```

Execute shell commands with pipes:

```bash
$ chronornd-job -command="sh" -n=2 -- -c "date | tee /tmp/timestamp.log"
```

## Design

### Features

- **Random Scheduling**: Jobs are scheduled at random times throughout the day
- **Sequential Execution**: Jobs are executed in chronological order
- **Daemon Mode**: Continuous operation with automatic queue replenishment
- **Signal Handling**: Graceful shutdown on SIGINT and SIGTERM
- **Command Arguments**: Full support for command-line arguments and pipes

### Job Queue Implementation

The daemon uses a simple queue-based design:

1. Generates the specified number of random execution times
2. Sorts them chronologically
3. Executes jobs sequentially from the queue
4. In daemon mode, generates new jobs when the queue is empty

### Logging

The daemon provides detailed logging of:

- Startup configuration
- Planned execution times
- Job execution status
- Error conditions

Example output:

```
2024/11/25 10:00:00 Starting chronornd-job (Command: ./backup.sh, Executions: 5, Mode: one-time)
2024/11/25 10:00:00 Planned execution times:
2024/11/25 10:00:00   11:23:45
2024/11/25 10:00:00   14:15:30
2024/11/25 10:00:00   16:48:12
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
