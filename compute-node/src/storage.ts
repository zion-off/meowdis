import type { ResultRow, Statement } from "./translator/types";

export interface StorageRPC {
  init(): Promise<void>;
  execBatch(statements: Statement[]): Promise<ResultRow[][]>;
  execPipeline(batches: { statements: Statement[] }[]): Promise<ResultRow[][][]>;
}
