# Start Next Ready Task

Start the next task that is ready to begin (unlocked and pending).

## Usage
```
/roady-task
```

## What it does
1. Runs `roady task ready` to find the next pending task
2. Starts the task with `roady task start <task-id>`
3. Reports the task details

## Example output
```
Starting task: task-user-auth
Title: Implement user authentication
Feature: authentication
```
