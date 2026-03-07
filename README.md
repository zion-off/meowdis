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

#### supported [redis 8.6](https://redis.io/docs/latest/commands/redis-8-6-commands/) commands (ambitious v1)

| category | commands                                                                          |
| -------- | --------------------------------------------------------------------------------- |
| strings  | `GET`, `SET`, `DEL`, `EXISTS`, `INCR`, `INCRBY`, `DECR`, `DECRBY`, `MGET`, `MSET` |
| expiry   | `EXPIRE`, `EXPIREAT`, `TTL`, `PTTL`, `PERSIST`                                    |
| hashes   | `HGET`, `HSET`, `HDEL`, `HGETALL`, `HEXISTS`, `HKEYS`, `HVALS`                    |
| lists    | `LPUSH`, `RPUSH`, `LPOP`, `RPOP`, `LRANGE`, `LLEN`                                |
| sets     | `SADD`, `SREM`, `SMEMBERS`, `SISMEMBER`, `SCARD`                                  |
| utility  | `PING`, `DBSIZE`, `FLUSHDB`, `KEYS`                                               |

### data structures

strings

```sql
CREATE TABLE strings (
    key       TEXT PRIMARY KEY,
    value     TEXT NOT NULL,
    expires_at INTEGER -- unix timestamp, NULL = no expiry
);
```

hashses

```sql
CREATE TABLE hashes (
    key       TEXT NOT NULL,
    field     TEXT NOT NULL,
    value     TEXT NOT NULL,
    expires_at INTEGER,
    PRIMARY KEY (key, field)
);
```

lists

```sql
CREATE TABLE lists (
    key       TEXT NOT NULL,
    index     REAL NOT NULL, -- float for O(1) prepend/append tricks
    value     TEXT NOT NULL,
    expires_at INTEGER,
    PRIMARY KEY (key, index)
);
```

sets

```sql
CREATE TABLE sets (
    key       TEXT NOT NULL,
    member    TEXT NOT NULL,
    expires_at INTEGER,
    PRIMARY KEY (key, member)
);
```
