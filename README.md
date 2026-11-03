# BLUIS

**BLUIS** is a Redis-compatible, in-memory key-value server written from scratch in Go, speaking the **RESP (Redis Serialization Protocol)** so any Redis client can talk to it.

```bash
go run -race .
redis-cli -p 6380 SET name Swarn   # OK
redis-cli -p 6380 GET name         # Swarn
```

---

## Features

- **RESP protocol** — decodes arrays of bulk strings from clients and encodes RESP replies (simple strings, errors, integers, bulk strings, null bulk strings).
- **Commands** — `PING`, `SET`, `GET`, `DEL`.
- **Concurrent clients** — every connection is served in its own goroutine.
- **Thread-safe storage** — a single shared store protected by `sync.RWMutex`.
- **Graceful error handling** — malformed RESP input gets an `-ERR protocol error: ...` reply and only that connection is dropped; other clients are unaffected.
- **Zero dependencies** — standard library only.

## Requirements

- Go **1.26.4+**
- Optional: `redis-cli` for interactive testing

## Quick start

```bash
# start the server (listens on localhost:6380)
go run -race .

# in another terminal
redis-cli -p 6380 PING
redis-cli -p 6380 SET name Swarn
redis-cli -p 6380 GET name
redis-cli -p 6380 DEL name
```

## Supported commands

| Command | Usage | Reply |
|---|---|---|
| `PING` | `PING` | `+PONG\r\n` (simple string) |
| `PING` | `PING <message>` | `$<len>\r\n<message>\r\n` (bulk string) |
| `SET` | `SET <key> <value>` | `+OK\r\n` |
| `GET` | `GET <key>` | value as bulk string, or `$-1\r\n` if missing |
| `DEL` | `DEL <key> [key ...]` | integer — number of keys removed (`:1\r\n`, `:0\r\n`) |

Unknown commands and wrong argument counts return `-ERR ...` replies.

## RESP wire examples

Any RESP client can drive BLUIS. Raw bytes over TCP:

```
→ *3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nSwarn\r\n
← +OK\r\n

→ *2\r\n$3\r\nGET\r\n$4\r\nname\r\n
← $5\r\nSwarn\r\n

→ *2\r\n$3\r\nGET\r\n$4\r\nnone\r\n
← $-1\r\n

→ *2\r\n$3\r\nDEL\r\n$4\r\nname\r\n
← :1\r\n
```

Values are length-prefixed, so **spaces, newlines and UTF-8 are safe**:

```
→ *3\r\n$3\r\nSET\r\n$5\r\nabout\r\n$11\r\nhello world\r\n
← +OK\r\n

→ *2\r\n$3\r\nGET\r\n$5\r\nabout\r\n
← $11\r\nhello world\r\n
```

Malformed input is rejected without disturbing other clients:

```
→ garbage\r\n
← -ERR protocol error: expected array header, got "garbage"\r\n
   (connection closed)
```

## Project structure

```
bluis/
├── main.go                     # TCP server, client loop, command handler
├── main_test.go                # handler + real-TCP integration tests
├── internals/
│   ├── resp/
│   │   ├── resp.go             # RESP parser and reply encoders
│   │   └── resp_test.go        # parser and encoder tests
│   └── storage/
│       └── storage.go          # mutex-protected in-memory key-value store
├── go.mod
└── README.md
```

Responsibilities stay separated:

- **TCP server** (`main.go`) — accepts connections and runs each client in a goroutine.
- **RESP parser** (`internals/resp`) — decodes incoming commands into `[]string`, encodes replies.
- **Command handler** (`handleCommand`) — dispatches `PING`/`SET`/`GET`/`DEL` to storage.
- **Storage** (`internals/storage`) — the shared key-value map guarded by `sync.RWMutex`.

## How it works

1. The accept loop hands every connection to `go handleClient(conn, store)`.
2. `resp.ReadCommand` reads the `*N` array header, then each `$len` bulk string **using `io.ReadFull` on exactly the declared byte length plus CRLF** — never `strings.Fields`, so values stay intact.
3. The resulting `[]string` is passed to `handleCommand`, which calls `store.Set/Get/Delete`.
4. The handler's reply is built with RESP encoders (`+`, `-`, `:`, `$`) and written back on the same connection.
5. The client loop repeats until the peer disconnects or sends malformed RESP.

## Testing

```bash
gofmt -l .          # formatting (clean)
go vet ./...        # static checks
go test ./...       # full suite
go test -race ./... # full suite under the race detector
```

The suite covers:

- RESP parsing: valid arrays, sequential commands on one reader, empty arrays, values with spaces, UTF-8 byte lengths.
- Malformed input: plain text, bad lengths, negative lengths, missing CRLF, truncated payloads (12 cases).
- All five reply encoders.
- Every command: `PING`, `SET`/`GET` round-trips, missing keys → `$-1\r\n`, `DEL` → `:1`/`:0`, arity and unknown-command errors.
- Real TCP: multiple commands on one connection, protocol-error handling, and **8 concurrent clients × 25 iterations** of `SET`/`GET`/`DEL` — verified clean under `-race`.

## Performance & concurrency notes

- Reads take an `RLock`, writes take a `Lock` — concurrent `GET`s never block each other.
- One goroutine per connection; no worker pools or channels in the hot path.
- The parser allocates only what it needs and caps array length (1M elements) and bulk size (512 MB) so a hostile header cannot exhaust memory.

## Future scope

BLUIS is feature-complete for its current goal (RESP + core key-value commands). Planned next:

- **Persistence** — `SAVE`/`BGSAVE` RDB-style snapshots and an AOF append-only log, with reload on boot.
- **Key expiry** — `EXPIRE`/`TTL`/`PEXPIRE`, lazy + periodic active deletion.
- **More data types** — `LPUSH`/`LRANGE` (lists), `HSET`/`HGET` (hashes), `SADD`/`SMEMBERS` (sets), `INCR`/`DECR`.
- **Bulk deletes & scans** — `DEL` across large key spaces, `KEYS`/`SCAN` with cursor-based iteration.
- **Server commands** — `INFO`, `DBSIZE`, `FLUSHDB`, `ECHO`, `SELECT`.
- **Authentication** — `AUTH` with a configured password.
- **Transactions** — `MULTI`/`EXEC`/`DISCARD` queueing.
- **Pub/Sub** — `SUBSCRIBE`/`PUBLISH` channel messaging.
- **Lua scripting** — `EVAL` with a sandboxed interpreter.
- **Replication** — primary/replica with `REPLICAOF` and partial resync.
- **Cluster mode** — hash-slot sharding across nodes.
- **CLI client** — a bundled `bluis-cli` REPL.
- **Benchmarks** — `redis-benchmark`-style load tests and CI perf tracking.

## License

See repository for license details.
