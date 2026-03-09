export const ErrWrongType = new Error(
  "WRONGTYPE Operation against a key holding the wrong kind of value"
);
export const ErrNotInteger = new Error("ERR value is not an integer or out of range");

export function errUnknownCommand(cmd: string): Error {
  return new Error(`ERR unknown command '${cmd.toLowerCase()}'`);
}

export function errWrongArgs(cmd: string): Error {
  return new Error(`ERR wrong number of arguments for '${cmd.toLowerCase()}' command`);
}
