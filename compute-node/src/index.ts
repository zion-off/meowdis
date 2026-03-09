import type { ResultRow, Translation } from "./translator/types";
import { translate } from "./translator";
import type { StorageRPC } from "./storage";

interface Env {
    STORAGE: DurableObjectNamespace;
    AUTH_TOKEN: string;
}

function jsonResponse(body: unknown, status = 200): Response {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

function textResponse(body: string, status: number): Response {
    return new Response(body, { status, headers: { "Content-Type": "text/plain" } });
}

function isStringArray(value: unknown): value is string[] {
    return Array.isArray(value) && value.every((item) => typeof item === "string");
}

function isStringArrayArray(value: unknown): value is string[][] {
    return Array.isArray(value) && value.every((item) => isStringArray(item));
}

function errorMessage(err: unknown): string {
    if (err instanceof Error) {
        return err.message;
    }
    return String(err);
}

export default {
    async fetch(request: Request, env: Env): Promise<Response> {
        const authHeader = request.headers.get("Authorization");
        if (!authHeader || !authHeader.startsWith("Bearer ")) {
            return textResponse("Unauthorized", 401);
        }
        const token = authHeader.slice("Bearer ".length);
        if (!env.AUTH_TOKEN || token !== env.AUTH_TOKEN) {
            return textResponse("Unauthorized", 401);
        }

        if (request.method !== "POST") {
            return textResponse("Method Not Allowed", 405);
        }

        let payload: unknown;
        try {
            payload = await request.json();
        } catch {
            return jsonResponse({ error: "ERR invalid request body" });
        }

        if (!Array.isArray(payload)) {
            return jsonResponse({ error: "ERR invalid request body" });
        }

        const id = env.STORAGE.idFromName("global");
        const stub = env.STORAGE.get(id) as DurableObjectStub & StorageRPC;

        if (isStringArray(payload)) {
            const cmd = payload;
            if (cmd.length === 1 && cmd[0].toUpperCase() === "INIT") {
                try {
                    await stub.init();
                    return jsonResponse({ result: "OK" });
                } catch (err) {
                    return jsonResponse({ error: errorMessage(err) });
                }
            }

            let translation: Translation;
            try {
                translation = translate(cmd);
            } catch (err) {
                return jsonResponse({ error: errorMessage(err) });
            }

            let results: ResultRow[][] = [];
            if (translation.statements.length > 0) {
                try {
                    results = await stub.execBatch(translation.statements);
                } catch (err) {
                    return jsonResponse({ error: errorMessage(err) });
                }
            }

            try {
                const mapped = translation.mapResult(results);
                return jsonResponse({ result: mapped });
            } catch (err) {
                return jsonResponse({ error: errorMessage(err) });
            }
        }

        if (!isStringArrayArray(payload)) {
            return jsonResponse({ error: "ERR invalid request body" });
        }

        const pipeline = payload;
        const results: unknown[] = new Array(pipeline.length);
        const translations: Translation[] = [];
        const indexMap: number[] = [];

        for (let i = 0; i < pipeline.length; i++) {
            try {
                const translation = translate(pipeline[i]);
                translations.push(translation);
                indexMap.push(i);
            } catch (err) {
                results[i] = { error: errorMessage(err) };
            }
        }

        let pipelineResults: ResultRow[][][] = [];
        if (translations.length > 0) {
            try {
                pipelineResults = await stub.execPipeline(
                    translations.map((translation) => ({ statements: translation.statements })),
                );
            } catch (err) {
                return jsonResponse({ error: errorMessage(err) });
            }
        }

        for (let i = 0; i < translations.length; i++) {
            const translation = translations[i];
            const resultIndex = indexMap[i];
            try {
                results[resultIndex] = translation.mapResult(pipelineResults[i] ?? []);
            } catch (err) {
                results[resultIndex] = { error: errorMessage(err) };
            }
        }

        return jsonResponse({ result: results });
    },
};
