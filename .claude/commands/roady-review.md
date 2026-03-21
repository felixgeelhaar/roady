# Check for Drift

Detect any discrepancies between the current implementation and the plan.

## Usage
```
/roady-review
```

## What it does
1. Runs `roady drift detect` to find implementation gaps
2. Runs `roady debt summary` for planning debt overview
3. Provides AI explanation of any drift found

## When to use
- Before starting new work
- After completing significant features
- During code review

## Example output
```
=== Drift Detection ===
✓ No spec drift
⚠ Plan drift: task-api-auth not started (expected: in_progress)
✓ No policy violations

=== Planning Debt ===
Debt Score: 72/100 (Good)
Sticky items: 2 (resolved in last 7 days)
```
