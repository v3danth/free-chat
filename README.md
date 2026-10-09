# Drift

A web chat for talking to strangers: make a card, walk into the room, knock
on someone for a private chat. Guests need no sign-up and are gone when they
leave. Every name draws its own pixel-art animal face.

Go, MySQL, WebSockets, and a plain HTML/JS frontend with no build step.

## Run it locally

Needs Go 1.26+, MySQL 8+ (Homebrew's works) and curl.

```sh
cp .env.example .env      # then fill in the MySQL login and JWT_SECRET
make db-create migrate    # create the databases and tables
make geoip                # optional: country flags
make start                # http://localhost:8080
```

`make start` checks MySQL, the schema and the port before it builds and runs
the server. `make help` lists every command.

Set `ADMIN_EMAIL` and `ADMIN_PASSWORD` in `.env` to create the admin on
startup. Sign in with them ("Sign in" tab) and open the control panel. Members
create their own accounts; moderators are added by the admin in the panel.

## Pages

| Path | What it is |
|---|---|
| `/` | Guest card, member sign-up and sign-in, the room, Here now, private chats, control panel |
| `/faces` | Type a name, see its face; rarity and the rules per animal |

## Docs

- [SPEC.md](SPEC.md): the HTTP and WebSocket API.
- [docs/naming.md](docs/naming.md): brand, domains and the search data
  behind them.
- [migrations/001_schema.sql](migrations/001_schema.sql): the database
  schema.

## Code layout

`cmd/server` wires everything together. Under `internal/`:

| Package | Job |
|---|---|
| `auth` | Guest sign-up, member login, JWT sessions, role checks |
| `user` | Profiles, cards and their validation |
| `websocket` | The live hub: rooms, presence, knock-and-door private chats |
| `message` | Message model, storage and the batched writer |
| `media` | Image uploads: re-encode and strip metadata |
| `moderation` | Reports, bans, mutes, banned words, audit log |
| `admin` | Control panel numbers and staff accounts |
| `avatar` | Pixel faces drawn from a name, and their endpoints |
| `filter`, `text` | The live banned-word filter |
| `block`, `ratelimit`, `geo`, `janitor` | Blocks, rate limits, country lookup, data retention |
| `apperr`, `httpx`, `config`, `database`, `id`, `mediapath` | Shared plumbing |

Inputs are parsed into known-good values where they enter (`Parse...`
functions); the code inside works only with those values, and views decide
what leaves.

## Tests

```sh
make test
```

`internal/avatar/testdata/golden.json` pins every face: if a change would
redraw existing faces, the test fails.
