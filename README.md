## meowdis

a free, self-hostable, upstash redis alternative

### goals

- upstash compatible api (works w upstash sdks)
- redis over http(s) / http compatible key-value store
- free storage backed by cloudflare durable objects
- low memory footprint with lambda go runtime

### high level architecture

#### compute-go / compute-ts (aws lambda / cloudflare worker)

- authenticated POST endpoints accept redis commands
- commands are parsed and transformed into a sqlite query
- sqlite query is passed on to the storage service
- results transformed back into redis response format
- responses are returned to the client
- bearer key authorization (static keys stored in env vars)

#### storage-sqlite (cloudflare durable objects)

- only accessible to compute services
- receives sqlite queries from them
- executes queries against built in sqlite database
- returns results back to the compute service

#### supported [redis 8.2](https://redis.io/docs/latest/commands/redis-8-2-commands/) commands

| category | commands                                                                          |
| -------- | --------------------------------------------------------------------------------- |
| strings  | `GET`, `SET`, `DEL`, `EXISTS`, `INCR`, `INCRBY`, `DECR`, `DECRBY`, `MGET`, `MSET` |
| expiry   | `EXPIRE`, `EXPIREAT`, `TTL`, `PTTL`, `PERSIST`                                    |
| hashes   | `HGET`, `HSET`, `HDEL`, `HGETALL`, `HEXISTS`, `HKEYS`, `HVALS`                    |
| lists    | `LPUSH`, `RPUSH`, `LPOP`, `RPOP`, `LRANGE`, `LLEN`                                |
| sets     | `SADD`, `SREM`, `SMEMBERS`, `SISMEMBER`, `SCARD`                                  |
| utility  | `PING`, `DBSIZE`, `FLUSHDB`, `KEYS`                                               |

### design decisions

#### data models

a central `keys` table owns the type and expiry for every key. type-specific tables store data only. this enforces the redis rule that a key can only have one type at a time — writing a key with a different type returns a `WRONGTYPE` error. expiry is deleted lazily on next read.

```sql
CREATE TABLE keys (
    key        TEXT PRIMARY KEY,
    type       TEXT NOT NULL CHECK(type IN ('string', 'hash', 'list', 'set')),
    expires_at INTEGER -- unix timestamp, NULL = no expiry
);

CREATE TABLE strings (
    key   TEXT PRIMARY KEY REFERENCES keys(key) ON DELETE CASCADE,
    value TEXT NOT NULL
);

CREATE TABLE hashes (
    key   TEXT NOT NULL REFERENCES keys(key) ON DELETE CASCADE,
    field TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (key, field)
);

CREATE TABLE lists (
    key   TEXT NOT NULL REFERENCES keys(key) ON DELETE CASCADE,
    index REAL NOT NULL, -- float for O(1) prepend/append tricks
    value TEXT NOT NULL,
    PRIMARY KEY (key, index)
);

CREATE TABLE sets (
    key    TEXT NOT NULL REFERENCES keys(key) ON DELETE CASCADE,
    member TEXT NOT NULL,
    PRIMARY KEY (key, member)
);
```

#### storage api

the compute layer sends a batch of sql statements to the storage durable object. the request uses `statements` for a single command or `pipeline` for multiple independent commands.

single command request:

```json
{
  "statements": [
    {
      "sql": "SELECT type, expires_at FROM keys WHERE key = ?",
      "params": ["foo"]
    },
    {
      "sql": "INSERT OR REPLACE INTO keys (key, type) VALUES (?, ?)",
      "params": ["foo", "string"]
    },
    {
      "sql": "INSERT OR REPLACE INTO strings (key, value) VALUES (?, ?)",
      "params": ["foo", "bar"]
    }
  ]
}
```

pipeline request:

```json
{
  "pipeline": [
    {
      "statements": [
        {
          "sql": "INSERT OR REPLACE INTO keys (key, type) VALUES (?, ?)",
          "params": ["foo", "string"]
        },
        {
          "sql": "INSERT OR REPLACE INTO strings (key, value) VALUES (?, ?)",
          "params": ["foo", "bar"]
        }
      ]
    },
    {
      "statements": [
        {
          "sql": "UPDATE strings SET value = CAST(value AS INTEGER) + 1 WHERE key = ? RETURNING value",
          "params": ["counter"]
        }
      ]
    }
  ]
}
```

the storage layer executes each item in its own `transactionSync` — pipeline items are independent and can fail separately:

```javascript
function execBatch(statements) {
  const results = [];
  this.ctx.storage.transactionSync(() => {
    for (const { sql, params } of statements) {
      results.push([...this.ctx.storage.sql.exec(sql, ...params)]);
    }
  });
  return results;
}

// single command
if (body.statements) return { results: execBatch(body.statements) };

// pipeline
if (body.pipeline)
  return {
    results: body.pipeline.map(({ statements }) => execBatch(statements)),
  };
```

single command response:

```json
{
  "results": [[{ "type": "string", "expires_at": null }], [], []]
}
```

pipeline response:

```json
{
  "results": [[[], []], [[{ "value": "1" }]]]
}
```

the compute layer picks whichever result set it needs. `DEL` from `keys` cascades automatically to all data tables.

#### translator

the translator is a pure function: `(command, args) → []Statement`. each command has its own handler that parses args and returns the appropriate sql statements.

options like `NX` and `EX` are parsed from the args and select different sql templates. `changes()` is used to chain dependent statements — the second statement is a no-op if the first affected zero rows.

supported `SET` options:

| option            | description                             |
| ----------------- | --------------------------------------- |
| `NX`              | only set if key does not already exist  |
| `XX`              | only set if key already exists          |
| `GET`             | return the old value before setting     |
| `EX seconds`      | expire after n seconds                  |
| `PX milliseconds` | expire after n milliseconds             |
| `EXAT timestamp`  | expire at unix timestamp (seconds)      |
| `PXAT timestamp`  | expire at unix timestamp (milliseconds) |
| `KEEPTTL`         | retain the existing expiry              |

not supported: `IFEQ`, `IFNE`, `IFDEQ`, `IFDNE` (require hash digest computation outside sqlite).

supported `EXPIRE` / `EXPIREAT` options:

| option | description                                           |
| ------ | ----------------------------------------------------- |
| `NX`   | only set expiry if key has no expiry                  |
| `XX`   | only set expiry if key already has an expiry          |
| `GT`   | only set expiry if new expiry is greater than current |
| `LT`   | only set expiry if new expiry is less than current    |

supported `LPOP` / `RPOP` options:

| option  | description                               |
| ------- | ----------------------------------------- |
| `count` | number of elements to pop (defaults to 1) |
