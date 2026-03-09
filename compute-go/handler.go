package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"compute/translator"
)

func encodeResult(v any) any {
	switch val := v.(type) {
	case string:
		return base64.StdEncoding.EncodeToString([]byte(val))
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = encodeResult(item)
		}
		return out
	default:
		return val
	}
}

func decodeJSON(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	return decoder.Decode(target)
}

func coerceStringSlice(values []any) ([]string, bool) {
	if len(values) == 0 {
		return []string{}, true
	}

	result := make([]string, len(values))
	for i, value := range values {
		switch v := value.(type) {
		case string:
			result[i] = v
		case json.Number:
			result[i] = v.String()
		case float64:
			result[i] = fmt.Sprintf("%v", v)
		default:
			return nil, false
		}
	}

	return result, true
}

func handler(w http.ResponseWriter, r *http.Request) {
	base64Encoding := r.Header.Get("Upstash-Encoding") == "base64"

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "ERR failed to read request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var singleRaw []any
	if err := decodeJSON(body, &singleRaw); err == nil {
		single, ok := coerceStringSlice(singleRaw)
		if !ok {
			goto pipelineCheck
		}
		if len(single) == 1 && strings.ToUpper(single[0]) == "INIT" {
			_, err := storagePost(map[string]any{"init": true})
			if err != nil {
				json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
				return
			}
			result := any("OK")
			if base64Encoding {
				result = encodeResult(result)
			}
			json.NewEncoder(w).Encode(map[string]any{"result": result})
			return
		}

		translation, err := translator.Translate(single)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		results, err := execStatements(translation.Statements)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		mapped, err := translation.MapResult(results)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		result := mapped
		if base64Encoding {
			result = encodeResult(result)
		}
		json.NewEncoder(w).Encode(map[string]any{"result": result})
		return
	}

pipelineCheck:

	var pipelineRaw [][]any
	if err := decodeJSON(body, &pipelineRaw); err == nil {
		pipeline := make([][]string, len(pipelineRaw))
		for i, cmdRaw := range pipelineRaw {
			cmd, ok := coerceStringSlice(cmdRaw)
			if !ok {
				pipeline = nil
				break
			}
			pipeline[i] = cmd
		}
		if pipeline == nil {
			goto invalidBody
		}
		results := make([]any, len(pipeline))
		translations := make([]translator.Translation, 0, len(pipeline))
		indexMap := make([]int, 0, len(pipeline))
		for i, cmd := range pipeline {
			translation, err := translator.Translate(cmd)
			if err != nil {
				results[i] = map[string]any{"error": err.Error()}
				continue
			}
			translations = append(translations, translation)
			indexMap = append(indexMap, i)
		}

		var pipelineResults [][][]map[string]any
		if len(translations) > 0 {
			batches := make([][]translator.Statement, len(translations))
			for i, translation := range translations {
				batches[i] = translation.Statements
			}
			var err error
			pipelineResults, err = execPipeline(batches)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
				return
			}
		}

		for i, translation := range translations {
			mapped, err := translation.MapResult(pipelineResults[i])
			if err != nil {
				results[indexMap[i]] = map[string]any{"error": err.Error()}
				continue
			}
			if base64Encoding {
				results[indexMap[i]] = encodeResult(mapped)
				continue
			}
			results[indexMap[i]] = mapped
		}

		json.NewEncoder(w).Encode(map[string]any{"result": results})
		return
	}

invalidBody:
	json.NewEncoder(w).Encode(map[string]any{"error": "ERR invalid request body"})
}
