package main

import (
	"encoding/json"
	"io"
	"net/http"

	"compute/translator"
)

func handler(w http.ResponseWriter, r *http.Request) {
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

	var single []string
	if err := json.Unmarshal(body, &single); err == nil {
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
		json.NewEncoder(w).Encode(map[string]any{"result": mapped})
		return
	}

	var pipeline [][]string
	if err := json.Unmarshal(body, &pipeline); err == nil {
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
			results[indexMap[i]] = mapped
		}

		json.NewEncoder(w).Encode(map[string]any{"result": results})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"error": "ERR invalid request body"})
}
