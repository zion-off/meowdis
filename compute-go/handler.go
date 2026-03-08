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
		statements, err := translator.Translate(single)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		results, err := execStatements(statements)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"result": results})
		return
	}

	var pipeline [][]string
	if err := json.Unmarshal(body, &pipeline); err == nil {
		var batches [][]translator.Statement
		for _, cmd := range pipeline {
			statements, err := translator.Translate(cmd)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			batches = append(batches, statements)
		}
		results, err := execPipeline(batches)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"result": results})
		return
	}

	http.Error(w, "ERR invalid request body", http.StatusBadRequest)
}
