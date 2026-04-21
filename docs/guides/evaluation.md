# Evaluate your first agent — `beluga eval`

**You will build:** a hand-authored dataset, run it against your scaffolded agent in mock mode (zero API cost), interpret the populated `eval-report.json`, switch to a real LLM for a full-quality pass, and wire it into CI.
**Prerequisites:** Go 1.25+, a working `beluga` CLI on `$PATH` (`go install github.com/lookatitude/beluga-ai/v2/cmd/beluga@latest`), and a scaffolded project (`beluga init …`). If you are brand new to Beluga, work through [Build Your First Agent](./first-agent.md) and [Dev Loop](./dev-loop.md) first.
**Related:** [CLI reference — `beluga eval`](../reference/cli.md#beluga-eval), [Architecture Overview — Layer 7](../architecture/01-overview.md#layer-7--application), [DOC-14 Observability](../architecture/14-observability.md).

`beluga eval` is the fourth and final slice of the DX-1 CLI epic. It lets you measure whether your agent is actually *good* — whether responses match expected outputs, whether a prompt change made things better or worse, whether a model swap regressed behaviour on a curated test set. The CLI builds your scaffolded binary with the existing `devloop.BuildBinary` pipeline, exec's it once per dataset row, reads the populated sample back from stdout, scores each configured metric, renders a stdout summary, writes `eval-report.json`, and exits with an aggregate pass/fail code suitable for CI gating.

## 1. Hand-author a dataset

A dataset is a plain JSON file. The `basic` template ships one at `.beluga/eval.smoke.json`; to start from scratch, create `my-eval.json`:

```json
{
  "name": "my-smoke",
  "samples": [
    {
      "Input": "Please echo: hello",
      "ExpectedOutput": "hello",
      "Turns": [
        {"Role": "assistant", "ToolCalls": [{"ID": "call_1", "Name": "echo", "Arguments": "{\"message\":\"hello\"}"}]},
        {"Role": "tool", "Content": "hello"},
        {"Role": "assistant", "Content": "hello"}
      ],
      "ExpectedTools": ["echo"]
    },
    {
      "Input": "Please echo: world",
      "ExpectedOutput": "world",
      "Turns": [
        {"Role": "assistant", "ToolCalls": [{"ID": "call_1", "Name": "echo", "Arguments": "{\"message\":\"world\"}"}]},
        {"Role": "tool", "Content": "world"},
        {"Role": "assistant", "Content": "world"}
      ],
      "ExpectedTools": ["echo"]
    }
  ]
}
```

Three fields matter per row:

- **`Input`** — the prompt handed to `agent.Invoke`.
- **`ExpectedOutput`** — the ground-truth reference answer. An empty string disables the `exact_match` metric for that row.
- **`Turns`** (optional) — a pre-recorded assistant trajectory used by the mock LLM provider to replay deterministic behaviour without an API key. Each assistant turn maps to one mock fixture via `mock.FixturesFromTurns`. You only need `Turns` for mock-mode smoke evals; real-provider runs ignore them.

The canonical JSON Schema is embedded in the binary:

```bash
beluga eval schema > eval-dataset.schema.json
```

Point your editor's JSON-Schema setting at that file to get autocomplete and validation as you author datasets.

## 2. Run the mock-mode smoke eval

The scaffolded `Makefile` ships an `eval-ci` target that pins the five canonical env vars for a deterministic, zero-cost smoke:

```bash
make eval-ci
```

Which expands to:

```bash
BELUGA_LLM_PROVIDER=mock BELUGA_DETERMINISTIC=1 BELUGA_SEED=42 OTEL_SDK_DISABLED=true \
  go run github.com/lookatitude/beluga-ai/v2/cmd/beluga eval .beluga/eval.smoke.json
```

What happens under the hood:

1. The CLI loads your dataset and `.beluga/eval.yaml` (metrics, row_timeout, parallel).
2. It builds your `main.go` via `devloop.BuildBinary`.
3. For each row it exec's the binary once with `BELUGA_ENV=eval` + `BELUGA_EVAL_SAMPLE_JSON=<row-json>`.
4. The scaffolded `main.go`'s eval-mode branch reads the row, seeds a `mock.ChatModel` with `mock.FixturesFromTurns(sample.Turns)`, invokes the agent, populates `sample.Output` + `Metadata["latency_ms"]`, and emits the populated sample JSON on stdout. The first line is the protocol probe `{"beluga_eval_protocol":1}`; the second line is the populated sample.
5. The CLI scores each configured metric (`exact_match`, `latency`) against the populated sample.
6. It renders a stdout table, writes `eval-report.json` to the project root, and exits.

```
IDX  ROW_ID  INPUT                                 OUTPUT  EXPECTED  exact_match  latency  ERROR
0    …       Please echo: hello                    hello   hello     1.00         1.00
1    …       Please echo: world                    world   world     1.00         1.00
2    …       Please echo: beluga                   beluga  beluga    1.00         1.00

run_id:  c1c…
dataset: sample-smoke (.beluga/eval.smoke.json)
samples: 3
duration: 420ms
aggregate:
  exact_match: 1.0000
  latency: 1.0000
```

`beluga eval` exits `0` when no row produces an exec error. Per-row metric failures are reported in the aggregate but do not flip the exit code — the convention is that `make eval-ci` exits green iff the mock trajectory is replayable end-to-end; aggregate-level pass-rate gating belongs in your CI assertion step (see §5).

## 3. Inspect `eval-report.json`

Every run writes `eval-report.json` at the project root — a stable location for CI artefact upload. Structure:

```json
{
  "run_id": "c1c…",
  "dataset": "sample-smoke",
  "dataset_path": ".beluga/eval.smoke.json",
  "started_at": "2026-04-22T11:02:01Z",
  "duration": "420ms",
  "samples": [
    {
      "index": 0,
      "row_id": "…",
      "input": "Please echo: hello",
      "output": "hello",
      "expected": "hello",
      "scores": {"exact_match": 1.0, "latency": 1.0}
    }
  ],
  "aggregate": {"exact_match": 1.0, "latency": 1.0},
  "errors": []
}
```

The `run_id` is the join key for traces + metrics + artefact. When OTel is enabled (see §6) the same UUID appears on the `eval.run` span as the `beluga.eval.run_id` resource attribute and on every sample in the `beluga.eval.metric.score` Histogram — you can pull the three together into a single view in any OTel backend that supports cross-signal joins.

## 4. Switch to a real provider

Mock mode replays `Turns`; real providers actually call the model. Swap by unsetting `BELUGA_LLM_PROVIDER`:

```bash
export OPENAI_API_KEY=sk-…
beluga eval my-eval.json
```

The scaffolded `main.go.tmpl`'s `buildEvalModel` helper defaults to the `openai` provider when `BELUGA_LLM_PROVIDER` is anything other than `mock`. To use a different provider, edit the `buildEvalModel` function in your scaffolded `main.go` — swap the blank-import, change the `llm.New("openai", …)` call to `llm.New("anthropic", …)` or another registered provider.

Real-provider runs produce per-sample non-determinism: run the same dataset twice against `gpt-4o-mini` and you may see different outputs. That is expected — the agent is sampling. For gate-worthy metrics on a real provider, prefer LLM-judge metrics (reserved for S4.5) over `exact_match`, or use a dataset small enough that human review of any regression is cheap.

## 5. Gate CI on the smoke eval

The scaffolded `.github/workflows/ci.yml` ships a Tier-1 `eval-smoke` job on every PR:

```yaml
eval-smoke:
  name: eval smoke (mock provider)
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version-file: go.mod
    - name: Make eval-ci
      run: make eval-ci
    - name: Upload eval report
      if: always()
      uses: actions/upload-artifact@v4
      with:
        name: eval-report
        path: eval-report.json
```

Budget: under 30 seconds on a cold-cache `ubuntu-latest` runner. Uses zero API keys. The uploaded `eval-report.json` is consumable by any downstream tool that reads the aggregated JSON (Braintrust push, dashboard publisher, regression bot).

The workflow also ships a **commented Tier-2 template** for real-provider eval. Uncomment it and add an `OPENAI_API_KEY` secret to enable a `workflow_dispatch`- or `schedule`-triggered full-quality pass:

```yaml
eval-real:
  name: eval full (real provider)
  runs-on: ubuntu-latest
  if: github.event_name == 'workflow_dispatch' || github.event_name == 'schedule'
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version-file: go.mod
    - name: Run real-provider eval
      env:
        OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
      run: go run github.com/lookatitude/beluga-ai/v2/cmd/beluga eval --max-rows 20 --max-cost 5.00 .beluga/eval.smoke.json
```

The template's structure — `if: github.event_name == 'workflow_dispatch' || github.event_name == 'schedule'` plus `secrets.OPENAI_API_KEY` — is the opt-in gate. Tier-1 stays the only evaluation on PRs until Tier-2 is explicitly uncommented; a GitHub Actions `schedule:` trigger alone is not an opt-in gate because it still runs without human review.

For PR-check annotations, pass `--format junit` and wire `dorny/test-reporter`:

```yaml
- name: Run make eval-ci with junit
  run: |
    BELUGA_LLM_PROVIDER=mock BELUGA_DETERMINISTIC=1 BELUGA_SEED=42 OTEL_SDK_DISABLED=true \
      go run github.com/lookatitude/beluga-ai/v2/cmd/beluga eval --format junit .beluga/eval.smoke.json
- uses: dorny/test-reporter@v1
  if: always()
  with:
    name: eval smoke
    path: eval-report.junit.xml
    reporter: java-junit
```

## 6. Observability

The framework-layer eval runner emits OTel spans per the [OpenTelemetry GenAI `gen_ai.evaluation.*` semantic convention](https://opentelemetry.io/docs/specs/semconv/gen-ai/) (Development status):

- **`eval.run`** span wraps the full CLI invocation. `gen_ai.operation.name = "eval"`; `beluga.eval.run_id` is a resource attribute.
- **`eval.row`** child span per dataset row. `gen_ai.operation.name = "eval"`; `beluga.eval.row_id` is a span attribute.
- **`gen_ai.evaluation.result`** events on the `eval.row` span, one per metric. Spec attributes: `gen_ai.evaluation.name`, `.score.value`, `.score.label`, `.explanation`.
- **`beluga.eval.metric.score`** OTel Histogram with two label dimensions: `beluga.eval.metric_name` and `beluga.eval.dataset`. Never `row_id` or `row_index` — cardinality rule.

`make eval-ci` sets `OTEL_SDK_DISABLED=true` so the stdout IPC stays clean and CI logs are quiet. Drop `OTEL_SDK_DISABLED=true` to collect spans locally with the `o11y` package's stdout exporter (`BELUGA_OTEL_STDOUT=1`) or forward them to an OTLP endpoint (`OTEL_EXPORTER_OTLP_ENDPOINT=…`). The `o11y.BootstrapFromEnv` contract from `beluga run` applies here unchanged.

## Common mistakes

- **Editing `Turns` and expecting a real provider to respect them.** `Turns` is consumed only by the mock LLM provider to derive fixtures. Real-provider runs ignore the field — the agent calls the real model and the model decides which tools to invoke. Test trajectories with the mock; measure quality with the real provider.
- **Setting `--parallel > 1` with `BELUGA_LLM_PROVIDER=mock`.** Per-row exec isolation keeps each row's mock queue independent (each row is a fresh process), so the CLI's default of `--parallel 1` is about preserving deterministic report ordering, not avoiding queue contamination. Still — keep it at `1` in mock mode unless you have a specific reason; the brief's `specialist-ai-ml-expert.md` §Q4 covers the design tradeoff.
- **Using `exact_match` as the quality gate on real-provider runs.** LLM sampling produces paraphrases that `exact_match` scores as 0. For real-provider quality gating, the LLM-judge metric family (reserved for S4.5) is the right tool; until then, use real-provider runs for observability only and gate CI on the mock smoke.
- **Pointing `go run ./cmd/beluga` inside the scaffolded project.** The scaffolded project's `go.mod` declares `github.com/lookatitude/beluga-ai/v2` as a `require`, so the correct invocation is `go run github.com/lookatitude/beluga-ai/v2/cmd/beluga eval …` (module path). `./cmd/beluga` is the framework-internal path and is not resolvable from a scaffolded project.
- **Running `beluga eval` against a pre-v2.13 scaffolded project.** Projects scaffolded before the DX-1 S4 changes lack the `BELUGA_ENV=eval` branch in `main.go`. The CLI rejects any child whose first stdout line is not the `{"beluga_eval_protocol":1}` probe within 5 seconds — add the branch manually (copy from the current template) or re-scaffold.

## Related reading

- [CLI reference — `beluga eval`](../reference/cli.md#beluga-eval) — exhaustive flag table and IPC contract.
- [Dev Loop guide](./dev-loop.md) — the `beluga run` / `dev` / `test` pipeline that shares the same `devloop.BuildBinary` + mock-provider infrastructure.
- [DOC-14 Observability](../architecture/14-observability.md) — the OTel GenAI convention landscape.
- [`eval/` package](../../eval/) — the framework-layer runner, metrics, and providers.
- [`cmd/beluga/eval/`](../../cmd/beluga/eval/) — the Layer 7 CLI adapter.
