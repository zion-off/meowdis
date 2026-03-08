// Package translator takes in Redis commands in a string array and translates them into SQLite queries.
package translator

import (
	"errors"
	"strings"
)

type Statement struct {
	SQL    string
	Params []any
}

type Translation struct {
	Statements []Statement
	MapResult  func(results [][]map[string]any) (any, error)
}

func Translate(cmd []string) (Translation, error) {
	if len(cmd) == 0 {
		return Translation{}, errors.New("ERR empty command")
	}
	switch strings.ToUpper(cmd[0]) {
	case "GET":
		return translateGet(cmd[1:])
	case "SET":
		return translateSet(cmd[1:])
	case "DEL":
		return translateDel(cmd[1:])
	case "EXISTS":
		return translateExists(cmd[1:])
	case "INCR":
		return translateIncr(cmd[1:])
	case "INCRBY":
		return translateIncrBy(cmd[1:])
	case "DECR":
		return translateDecr(cmd[1:])
	case "DECRBY":
		return translateDecrBy(cmd[1:])
	case "MGET":
		return translateMGet(cmd[1:])
	case "MSET":
		return translateMSet(cmd[1:])
	case "EXPIRE":
		return translateExpire(cmd[1:])
	case "EXPIREAT":
		return translateExpireAt(cmd[1:])
	case "TTL":
		return translateTTL(cmd[1:])
	case "PTTL":
		return translatePTTL(cmd[1:])
	case "PERSIST":
		return translatePersist(cmd[1:])
	case "HGET":
		return translateHGet(cmd[1:])
	case "HSET":
		return translateHSet(cmd[1:])
	case "HDEL":
		return translateHDel(cmd[1:])
	case "HGETALL":
		return translateHGetAll(cmd[1:])
	case "HEXISTS":
		return translateHExists(cmd[1:])
	case "HKEYS":
		return translateHKeys(cmd[1:])
	case "HVALS":
		return translateHVals(cmd[1:])
	case "LPUSH":
		return translateLPush(cmd[1:])
	case "RPUSH":
		return translateRPush(cmd[1:])
	case "LPOP":
		return translateLPop(cmd[1:])
	case "RPOP":
		return translateRPop(cmd[1:])
	case "LRANGE":
		return translateLRange(cmd[1:])
	case "LLEN":
		return translateLLen(cmd[1:])
	case "SADD":
		return translateSAdd(cmd[1:])
	case "SREM":
		return translateSRem(cmd[1:])
	case "SMEMBERS":
		return translateSMembers(cmd[1:])
	case "SISMEMBER":
		return translateSIsMember(cmd[1:])
	case "SCARD":
		return translateSCard(cmd[1:])
	case "PING":
		return translatePing(cmd[1:])
	case "DBSIZE":
		return translateDBSize(cmd[1:])
	case "FLUSHDB":
		return translateFlushDB(cmd[1:])
	case "KEYS":
		return translateKeys(cmd[1:])
	default:
		return Translation{}, errUnknownCommand(cmd[0])
	}
}

func deleteIfExpired(key string) Statement {
	return Statement{
		SQL:    "DELETE FROM keys WHERE key = ? AND expires_at IS NOT NULL AND expires_at <= unixepoch()",
		Params: []any{key},
	}
}
