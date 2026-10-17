# Engineering standards

Write code the way a senior Google engineer would: simple, readable,
tested, and fast where it matters. Follow Google's published guides and
open-source code as the reference:

- Go: Google Go Style Guide (google.github.io/styleguide/go), Effective Go,
  and the code in golang/go and google/* repos on GitHub.
- JavaScript: Google JavaScript Style Guide (google.github.io/styleguide/jsguide.html).
- Shell: Google Shell Style Guide (google.github.io/styleguide/shellguide.html).
- Design: Google's design-doc practice and SRE book. State the problem,
  the constraints and the trade-off before writing a large change.

When the codebase already has an idiom, match it. Consistency with the
surrounding code beats a style guide rule.

## Functions

- One function, one task. If its name needs "and", split it.
- Give every function a meaningful name chosen for how it is used. The
  name says what it does or what it returns, so a call site reads like
  a sentence:
  - Actions are verbs: `banUser`, `sendMessage`, `closeRoom`.
  - Functions that return a value are named for the value: `faceSeed`,
    `onlineCount`, `roomByID`. Go drops the `Get` prefix.
  - Booleans read as questions: `isBanned`, `hasPhoto`, `canModerate`.
  - Conversions say both ends: `profileToCard`, `parseDuration`.
  - Constructors are `New`/`NewX` in Go, `createX` in JS.
  - Name for the domain, not the mechanism: `expiredGuests`, not
    `filterList`. Package names are part of the name in Go, so write
    `avatar.Paint`, not `avatar.PaintAvatar`.
  - Short names only in tiny scopes (`i`, `r`, `w`); longer scope,
    longer name. No vague names: `data`, `info`, `handle`, `process`,
    `doStuff`, `helper`, `util`.
- The name and the code say *what* it does. Reading the body should be
  enough to know its behaviour without comments.
- The doc comment says *why* it exists: the problem it solves, the
  constraint it guards, or the caller that needs it. Never restate the code.

  ```go
  // Guests rejoin after a reconnect with the same name, so the face must
  // depend on the name alone, never on the session.
  func faceSeed(name string) uint64 { ... }
  ```

- Inline comments only for non-obvious reasons (a workaround, an
  invariant, a performance choice). No line-by-line narration.

## Functional style

- Prefer pure functions: inputs in, result out, no hidden state. Keep
  I/O (DB, network, clock, randomness) at the edges and pass results in.
- Do not mutate arguments; return new values. Mutate in place only on a
  measured hot path, and say so in a comment.
- Compose small functions instead of writing long ones. Pass behaviour
  as function values where it removes branching or duplication.
- Do not fight the language: in Go a plain `for` loop beats a homemade
  map/filter helper. Use `slices` and `maps` from the standard library.

## Reuse

- Search the codebase before writing a helper; extend the existing one.
- No copy-paste. A second copy of the same logic means extract it.
- Shared behaviour across handlers (auth, rate limits, logging, recovery,
  timing) goes in a wrapper, not repeated inside each handler.

## Wrappers and decorators

Use higher-order functions for cross-cutting concerns:

- Go HTTP: middleware `func(http.Handler) http.Handler`, chained in one place.
- Go calls: wrappers like `withTimeout(ctx, fn)`, `withRetry(fn)`, `timed(name, fn)`.
- JS: wrapper functions (`debounce`, `once`, `withErrorToast(fn)`).

A wrapper must be obviously correct and do one thing. Do not hide
business logic inside one.

## Logging

- Use `log/slog` for new code: `Debug` for tracing values while
  developing, `Info` for important lifecycle events (start, stop, config
  loaded, admin actions), `Warn` for recovered problems, `Error` for
  failures that need a human.
- Log structured key-value pairs, not formatted strings:
  `slog.Info("room joined", "room", id, "members", n)`.
- Log at the edge where the error is handled, once. Do not log and
  return the same error.
- Never log secrets or personal data: passwords, JWTs, API keys, session
  cookies, `MYSQL_PASSWORD`, raw IPs (log the hash), email addresses,
  message bodies, uploaded file contents. Log IDs instead.
- Do not log inside hot loops; log counts and durations after them.

## Performance (line by line)

Write every line as if it runs a million times a second, then measure
before optimising further.

- Know the cost of each line: allocations, copies, locks, syscalls,
  round trips.
- Preallocate when the size is known: `make([]T, 0, n)`, `strings.Builder` with `Grow`.
- Avoid allocations in hot paths: no `fmt.Sprintf` for simple joins, no
  conversions between `[]byte` and `string` that you don't need, reuse buffers.
- One DB round trip, not N: batch queries, select only the needed columns,
  index what you filter on.
- Hold locks for the shortest time; never do I/O while holding one.
- Bound everything: timeouts on every network call, caps on queue and
  input sizes.
- Prove optimisations with `go test -bench` and `pprof`; keep the
  benchmark. Readability wins on cold paths.
- JS: batch DOM writes, avoid layout thrash, debounce input handlers.

## Errors

- Handle every error. Wrap with context: `fmt.Errorf("load profile %d: %w", id, err)`.
- Return errors; do not panic outside `main` and init-time invariants.
- Validate input at the boundary (handlers), trust it inside.

## Before finishing a change

- `gofmt`, `go vet ./...`, `golangci-lint run ./...`, `go test ./...`.
- ESLint on changed `web/*.js`.
- Remove dead code and duplicates you created.
- No emojis anywhere in code, scripts, or their output.
- Commits are authored as v3danth with no AI attribution lines.
