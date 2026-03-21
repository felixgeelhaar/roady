# Full Project Status

Get a comprehensive overview of the project status including spec, plan, and drift.

## Usage
```
/roady-status
```

## What it does
1. Runs `roady status` for task overview
2. Checks for drift with `roady drift detect`
3. Shows AI usage with `roady usage`

## Example output
```
=== Project Status ===
Spec: User Authentication System
Features: 3 | Tasks: 12
Ready: 2 | In Progress: 1 | Done: 8 | Blocked: 1

=== Drift Check ===
No drift detected ✓

=== AI Usage ===
Tokens: 45,000 / 100,000 (45%)
```
