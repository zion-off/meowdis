import { errUnknownCommand } from "./errors";
import {
  translateDecr,
  translateDecrBy,
  translateDel,
  translateExists,
  translateGet,
  translateIncr,
  translateIncrBy,
  translateMGet,
  translateMSet,
  translateSet,
} from "./strings";
import {
  translateExpire,
  translateExpireAt,
  translatePersist,
  translatePTTL,
  translateTTL,
} from "./expiry";
import {
  translateHDel,
  translateHExists,
  translateHGet,
  translateHGetAll,
  translateHKeys,
  translateHSet,
  translateHVals,
} from "./hashes";
import {
  translateLLen,
  translateLPop,
  translateLPush,
  translateLRange,
  translateRPop,
  translateRPush,
} from "./lists";
import {
  translateSAdd,
  translateSCard,
  translateSIsMember,
  translateSMembers,
  translateSRem,
} from "./sets";
import { translateDBSize, translateFlushDB, translateKeys, translatePing } from "./utility";
import type { Translation } from "./types";

export function translate(cmd: string[]): Translation {
  if (cmd.length === 0) {
    throw new Error("ERR empty command");
  }

  switch (cmd[0].toUpperCase()) {
    case "GET":
      return translateGet(cmd.slice(1));
    case "SET":
      return translateSet(cmd.slice(1));
    case "DEL":
      return translateDel(cmd.slice(1));
    case "EXISTS":
      return translateExists(cmd.slice(1));
    case "INCR":
      return translateIncr(cmd.slice(1));
    case "INCRBY":
      return translateIncrBy(cmd.slice(1));
    case "DECR":
      return translateDecr(cmd.slice(1));
    case "DECRBY":
      return translateDecrBy(cmd.slice(1));
    case "MGET":
      return translateMGet(cmd.slice(1));
    case "MSET":
      return translateMSet(cmd.slice(1));
    case "EXPIRE":
      return translateExpire(cmd.slice(1));
    case "EXPIREAT":
      return translateExpireAt(cmd.slice(1));
    case "TTL":
      return translateTTL(cmd.slice(1));
    case "PTTL":
      return translatePTTL(cmd.slice(1));
    case "PERSIST":
      return translatePersist(cmd.slice(1));
    case "HGET":
      return translateHGet(cmd.slice(1));
    case "HSET":
      return translateHSet(cmd.slice(1));
    case "HDEL":
      return translateHDel(cmd.slice(1));
    case "HGETALL":
      return translateHGetAll(cmd.slice(1));
    case "HEXISTS":
      return translateHExists(cmd.slice(1));
    case "HKEYS":
      return translateHKeys(cmd.slice(1));
    case "HVALS":
      return translateHVals(cmd.slice(1));
    case "LPUSH":
      return translateLPush(cmd.slice(1));
    case "RPUSH":
      return translateRPush(cmd.slice(1));
    case "LPOP":
      return translateLPop(cmd.slice(1));
    case "RPOP":
      return translateRPop(cmd.slice(1));
    case "LRANGE":
      return translateLRange(cmd.slice(1));
    case "LLEN":
      return translateLLen(cmd.slice(1));
    case "SADD":
      return translateSAdd(cmd.slice(1));
    case "SREM":
      return translateSRem(cmd.slice(1));
    case "SMEMBERS":
      return translateSMembers(cmd.slice(1));
    case "SISMEMBER":
      return translateSIsMember(cmd.slice(1));
    case "SCARD":
      return translateSCard(cmd.slice(1));
    case "PING":
      return translatePing(cmd.slice(1));
    case "DBSIZE":
      return translateDBSize(cmd.slice(1));
    case "FLUSHDB":
      return translateFlushDB(cmd.slice(1));
    case "KEYS":
      return translateKeys(cmd.slice(1));
    default:
      throw errUnknownCommand(cmd[0]);
  }
}
