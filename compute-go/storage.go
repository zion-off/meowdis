package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"compute/translator"
)

type storageStatement struct {
	SQL    string `json:"sql"`
	Params []any  `json:"params"`
}

type storageBatch struct {
	Statements []storageStatement `json:"statements"`
}

type storageResponse struct {
	Results json.RawMessage `json:"results"`
	Error   string          `json:"error,omitempty"`
}

func toStorageStatements(stmts []translator.Statement) []storageStatement {
	out := make([]storageStatement, len(stmts))
	for i, s := range stmts {
		out[i] = storageStatement{SQL: s.SQL, Params: s.Params}
	}
	return out
}

func storagePost(body any) (json.RawMessage, error) {
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("STORAGE_ENDPOINT not set")
	}
	token := os.Getenv("STORAGE_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("STORAGE_TOKEN not set")
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sr storageResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	if sr.Error != "" {
		return nil, fmt.Errorf("%s", sr.Error)
	}

	return sr.Results, nil
}

func execStatements(stmts []translator.Statement) ([][]map[string]any, error) {
	raw, err := storagePost(map[string]any{
		"statements": toStorageStatements(stmts),
	})
	if err != nil {
		return nil, err
	}

	var results [][]map[string]any
	return results, json.Unmarshal(raw, &results)
}

func execPipeline(batches [][]translator.Statement) ([][][]map[string]any, error) {
	pipeline := make([]storageBatch, len(batches))
	for i, stmts := range batches {
		pipeline[i] = storageBatch{Statements: toStorageStatements(stmts)}
	}

	raw, err := storagePost(map[string]any{
		"pipeline": pipeline,
	})
	if err != nil {
		return nil, err
	}

	var results [][][]map[string]any
	return results, json.Unmarshal(raw, &results)
}
