# evals

Regression tests over Roady's planning pipeline.

The heuristic planner is deterministic: a given `spec.yaml` produces the same
plan every time. This package locks that contract by pairing each fixture
spec with a `golden.yaml` describing the exact tasks (IDs, feature IDs,
dependency edges) the planner is expected to produce.

## Running

```bash
go test ./evals/...                       # default (no API keys, free)
go test -tags evals_ai ./evals/...        # opt-in real-provider matrix
```

Runs as part of `go test ./...` in CI; nothing extra to wire up.

The `evals_ai` build tag enables `ai_real_providers_test.go`, which loops
the AI planner through the real Anthropic / OpenAI / Gemini / Ollama
providers when their respective env vars are set:

| Provider | Env var |
| --- | --- |
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI | `OPENAI_API_KEY` |
| Gemini | `GEMINI_API_KEY` |
| Ollama (local daemon) | `ROADY_EVAL_ENABLE_OLLAMA` (any value) |

Providers without their env var set are **skipped, not failed**, so the
suite runs cleanly with whatever keys you have on hand.

## Layout

```
evals/
├── README.md
├── runner_test.go              # heuristic planner -> golden plan check
├── ai_runner_test.go           # AI planner pipeline check via programmable mock
├── drift_runner_test.go        # synthetic spec/plan/state divergence corpus
└── fixtures/
    ├── cli-tool/               # single-feature CLI
    │   ├── spec.yaml
    │   └── golden.yaml
    ├── web-api/                # auth + tasks REST API
    │   ├── spec.yaml
    │   └── golden.yaml
    └── multi-feature/          # mixes empty-feature fallback + multi-req feature
        ├── spec.yaml
        └── golden.yaml
```

The AI runner uses a programmable mock provider that returns canned JSON
shaped like a real model output. It locks the AI pipeline contract:
parsing, `Origin=ai` tagging, missing-feature backfill (which produces a
`Origin=heuristic` task), and policy gating. Real-provider runs stay out of
CI to keep the suite free; an opt-in `EVAL_AI=1` build tag will gate them
later.

## Adding a new fixture

1. Create a directory under `fixtures/`.
2. Drop a `spec.yaml` representing the input (same schema as `.roady/spec.yaml`).
3. Run `go test ./evals/... -run TestHeuristicPlannerMatchesGoldens/<your-fixture>`
   once and copy the failure message — it lists the produced task IDs and
   dependency edges.
4. Hand-write `golden.yaml` to match the produced output **only after** you
   have reviewed it and confirmed the planner did the right thing for this
   spec.

The same workflow applies when a planner change is intentional: rerun the
test, inspect the diff, and update goldens deliberately. A silent diff is
the failure mode this harness exists to prevent.

## Updating an existing golden

Treat goldens like checked-in expectations — never auto-regenerate without a
human reading the diff. The recommended flow:

```bash
# 1. Run the failing eval to see the diff in test output.
go test ./evals/... -v -run TestHeuristicPlannerMatchesGoldens/<fixture>

# 2. If the new output is correct, edit the golden by hand to match.
$EDITOR evals/fixtures/<fixture>/golden.yaml

# 3. Re-run; tests pass; commit fixture + golden + planner change together.
```

## Scope and policy

| Area | Today | Future |
| --- | --- | --- |
| Heuristic planner | Locked by golden tests | — |
| AI planner pipeline | Locked via programmable mock; opt-in real-provider matrix via `-tags evals_ai` | — |
| Drift detector | Synthetic divergence corpus (precision + recall) | Expand to code/policy drift scenarios |
| Provider parity | Not yet | Same spec across providers; structural agreement check |

Keep fixture inputs small and motivated by a real planner edge case
(empty-feature fallback, internal dependency, cross-feature edge). Big
fixtures slow the suite without adding signal.
