# Playbook: measure, model, simulate

A general method for making an app fast and correct. It is not tied to
Drift; copy it into any project. CLAUDE.md sets the coding rules, this
file sets the order of work.

The loop is always the same:

1. Model the behaviour, so bugs have nowhere to hide.
2. Simulate real users, so bugs show up without waiting for them.
3. Measure, so you fix what is slow and not what looks slow.
4. Change one thing, measure again, keep the benchmark.

## 1. Model UI behaviour as state machines

Any piece of UI state that changes over time (a chat, a request, a
connection, a wizard, a page) is a machine: a table of
`state -> event -> next state`. A loose status string that several
handlers set directly is the bug, even before it misbehaves.

Rules:

- One table per thing that changes. Keep it data, not a chain of `if`s:
  ```js
  const chat = {
    none:     { knock: 'outgoing', knocked: 'incoming', left: 'gone' },
    incoming: { reply: 'open', decline: 'none', left: 'gone' },
    // ...
  };
  ```
- One tiny helper applies moves (`createMachine(name, table, initial)`
  with `send(event)` returning whether the move was allowed). No library
  unless the project already uses one.
- A move that is not in the table is refused and reported. In
  development it is loud (console error plus an on-page toast), so every
  undesigned path becomes visible the first time it happens.
- Rendering reads the state; it never decides it. Handlers send events;
  they never assign state.
- The server stays the source of truth. Give every machine one `sync`
  event, allowed from any state, that replays the server's view after a
  reconnect or reload.
- Decide what survives a reload for each state, on purpose, and write it
  down. "Comes back empty after refresh" is a missing state, not a
  missing feature.
- Draw each table in a design doc (Mermaid `stateDiagram-v2`) next to
  the problem it solves. Walk every state and ask: how does the user get
  out of here? A state with no exit, or an exit nobody designed, is the
  bug list.
- Unit-test the tables as pure data: every listed move lands where the
  doc says, and a sample of unlisted moves is refused.

## 2. Simulate users to find bugs

Manual clicking finds the happy path. Scripted users find the rest.

- Build a simulator (`cmd/simulate`): bots that join, act, reconnect,
  leave and come back, driven by a small decision function that is
  pure and unit-tested.
- Run it against a dev server with the machines in loud mode. Every
  refused move it triggers is a real bug with a reproducible script.
- Include the rude cases: two users acting at the same instant, reload
  mid-action, the other side leaving, a blocked user, a slow network.
- Keep the simulator in the repo; rerun it after every behaviour change.

## 3. Measure before changing anything

Never optimise from intuition. Get a number, find where the time goes,
then change code.

Tools, from cheapest to most realistic:

- Micro benchmarks (`go test -bench=. -benchmem`) on the hot functions:
  encoding, the join path, caches, query builders. Report ns/op,
  B/op and allocs/op.
- A load tester (`cmd/loadtest`) that reports p50/p95/p99 latency and
  has a hammer mode to find the breaking point at realistic scale
  (for example 1000 users online).
- Profiles: `pprof` on a loopback-only address behind an env var
  (`PPROF_ADDR`), never on the public listener. CPU, heap, allocs,
  mutex and block profiles, captured under load.
- Browser: the Performance panel for long tasks, layout thrash and
  paint; the Network panel for payload size and request count.

Write the baseline down before the change. Every performance claim in a
commit carries before and after numbers from the same machine:
`arrival at 1000 online 1.69 ms -> 0.17 ms`. Keep the benchmark that
proved it, so a regression shows up in `go test -bench`.

## 4. System design levers

Look for these in order of usual payoff. Each one is a known pattern;
apply it only where a profile points.

Do less work:

- Cache the encoded result, not just the data. If every arrival
  re-encodes N records, cache each encoded record and splice them in.
- Skip no-op writes: compare before writing, and skip the re-read when
  nothing changed.
- Query only what is needed now (only online users, only the visible
  page, only the needed columns).
- Bound caches with an eviction policy. Approximate LRU (sample a few
  keys, evict the oldest) is simpler and nearly as good as exact LRU.

Fewer round trips:

- One DB round trip, not N. Batch, join, or `IN (...)`.
- Coalesce small network writes into one frame or one flush.
- Interpolate DB params client-side when the driver supports it, to
  drop the prepare round trip.
- Match the idle connection pool to the open limit so bursts do not
  redial.

Fewer bytes and allocations:

- Compress only above a size threshold (small frames cost more to
  compress than to send).
- Pool reusable buffers (`sync.Pool`) on hot write paths.
- Serve static files from memory, pre-gzipped, with ETag; mark
  fingerprinted assets immutable.

Stay up under load:

- Timeouts on every dial, read and write. Caps on every queue, body and
  input.
- No I/O while holding a lock. Copy what you need, unlock, then write.
- One slow consumer must never stall the others (per-client send
  queues, drop or disconnect when full).

Client:

- Update the DOM incrementally; never rebuild a whole pane on one
  change.
- Batch DOM writes per frame; read layout before writing it.
- Animate only `transform` and `opacity` so the compositor does the
  work.
- Cache derived values (parsed names, formatted notes) and invalidate
  them on the event that changes them.

## 5. Order of work for a change

1. Write the problem, constraints and trade-off (a short design doc for
   anything larger than a fix).
2. Draw or update the state tables the change touches.
3. Add or update the benchmark and the simulator script for it.
4. Record the baseline numbers.
5. Make the change. One lever at a time.
6. Rerun benchmarks, load test and simulator. Keep the change only if
   the numbers moved and no refused moves appeared.
7. Run the checks in CLAUDE.md (gofmt, vet, golangci-lint, tests,
   ESLint).
8. Commit with the measured before and after in the message.

## 6. What not to do

- Do not optimise a cold path at the cost of readability.
- Do not add a cache without a bound and an invalidation rule.
- Do not claim a speedup without numbers, or keep one whose benchmark
  was deleted.
- Do not let a handler set UI state directly "just this once".
- Do not expose pprof, simulators or load tools on a public address.
