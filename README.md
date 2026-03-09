## meowdis

a free, serverless, self-hostable redis clone backed by cloudflare durable objects. drop-in replacement for upstash redis.

### features

- upstash compatible api (works w upstash sdks)
- redis over http(s) / http compatible key-value store
- pipelining support
- free storage backed by cloudflare durable objects
- low memory footprint <3 lambda go runtime

### supported [redis 8.2](https://redis.io/docs/latest/commands/redis-8-2-commands/) commands

| category | commands                                                                          |
| -------- | --------------------------------------------------------------------------------- |
| strings  | `GET`, `SET`, `DEL`, `EXISTS`, `INCR`, `INCRBY`, `DECR`, `DECRBY`, `MGET`, `MSET` |
| expiry   | `EXPIRE`, `EXPIREAT`, `TTL`, `PTTL`, `PERSIST`                                    |
| hashes   | `HGET`, `HSET`, `HDEL`, `HGETALL`, `HEXISTS`, `HKEYS`, `HVALS`                    |
| lists    | `LPUSH`, `RPUSH`, `LPOP`, `RPOP`, `LRANGE`, `LLEN`                                |
| sets     | `SADD`, `SREM`, `SMEMBERS`, `SISMEMBER`, `SCARD`                                  |
| utility  | `PING`, `DBSIZE`, `FLUSHDB`, `KEYS`                                               |

### how it works (basically)

- a compute service (compute-node or compute-go) exposes a POST endpoint
- accepts upstash redis commands in the request body
- translates the command into sqlite queries
- forwards the queries to the storage service, a durable object instance
- durable object executes the queries against its sqlite database and returns the result
- compute service returns the result to the client

### setup

**1.** deploy the storage layer

[![Deploy to Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/zion-off/meowdis&dir=durable-object)

**2.** deploy the compute layer -- set the following secret when prompted

| secret       | description                                                                                               |
| ------------ | --------------------------------------------------------------------------------------------------------- |
| `AUTH_TOKEN` | a random secret string -- generate one at [jwtsecretkeygenerator.com](https://jwtsecretkeygenerator.com/) |

[![Deploy to Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/zion-off/meowdis&dir=compute-node)

**3.** initialise the database

```bash
curl https://your-compute-endpoint \
  -H "Authorization: Bearer your-token" \
  -d '["INIT"]'
```

**4.** verify it's working

```bash
curl https://your-compute-endpoint \
  -H "Authorization: Bearer your-token" \
  -d '["PING"]'                       # {"result":"PONG"}
```

> **prefer go?** skip step 2 and deploy `compute-go` to aws lambda instead. configure these environment variables:
>
> | variable           | description                                                                                               |
> | ------------------ | --------------------------------------------------------------------------------------------------------- |
> | `AUTH_TOKEN`       | a random secret string -- generate one at [jwtsecretkeygenerator.com](https://jwtsecretkeygenerator.com/) |
> | `STORAGE_ENDPOINT` | url of your deployed durable object worker                                                                |
> | `STORAGE_TOKEN`    | the `AUTH_TOKEN` value you chose                                                                          |

### usage examples

**python**

```python
from upstash_redis import Redis

redis = Redis(url="https://compute-endpoint", token="your-token")

redis.ping()                          # 'PONG'
redis.set("name", "clairo")           # 'OK'
redis.get("name")                     # 'clairo'
```

**node**

```typescript
import { Redis } from "@upstash/redis";

const redis = new Redis({
  url: "https://compute-endpoint",
  token: "your-token",
});

await redis.ping(); // 'PONG'
await redis.set("name", "clairo"); // 'OK'
await redis.get("name"); // 'clairo'
```

**rest** (see
[upstash rest api docs](https://upstash.com/docs/redis/features/restapi))

```bash
curl https://compute-endpoint \
  -H "Authorization: Bearer your-token" \
  -d '["PING"]'                       # {"result":"PONG"}

curl https://compute-endpoint \
  -H "Authorization: Bearer your-token" \
  -d '["SET", "name", "clairo"]'      # {"result":"OK"}

curl https://compute-endpoint \
  -H "Authorization: Bearer your-token" \
  -d '["GET", "name"]'                # {"result":"clairo"}
```
