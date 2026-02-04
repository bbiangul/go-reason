package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/brunobiangulo/goreason"
)

type handler struct {
	engine goreason.Engine
}

func newHandler(e goreason.Engine) *handler {
	return &handler{engine: e}
}

// POST /ingest
// Accepts multipart file upload or JSON with file path.
func (h *handler) handleIngest(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	// Try multipart upload first
	if err := r.ParseMultipartForm(100 << 20); err == nil { // 100MB max
		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()

			// Sanitise filename to prevent path traversal.
			safeName := filepath.Base(header.Filename)

			tmpDir := os.TempDir()
			tmpPath := filepath.Join(tmpDir, safeName)
			dst, err := os.Create(tmpPath)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to process file")
				slog.Error("creating temp file", "error", err)
				return
			}
			if _, err := io.Copy(dst, file); err != nil {
				dst.Close()
				writeError(w, http.StatusInternalServerError, "failed to save file")
				slog.Error("saving uploaded file", "error", err)
				return
			}
			dst.Close()
			defer os.Remove(tmpPath)

			docID, err := h.engine.Ingest(ctx, tmpPath)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "ingestion failed")
				slog.Error("ingest error", "error", err)
				return
			}

			writeJSON(w, http.StatusOK, map[string]interface{}{
				"document_id": docID,
				"filename":    safeName,
			})
			return
		}
	}

	// Try JSON body with path
	var req struct {
		Path    string            `json:"path"`
		Options map[string]string `json:"options,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: expected multipart file or JSON with 'path'")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	// Validate that path is a real file (prevents directory traversal probing).
	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusBadRequest, "path must be an existing file")
		return
	}

	var opts []goreason.IngestOption
	if req.Options != nil {
		if _, ok := req.Options["force"]; ok {
			opts = append(opts, goreason.WithForceReparse())
		}
		if method, ok := req.Options["parse_method"]; ok {
			opts = append(opts, goreason.WithParseMethod(method))
		}
	}

	docID, err := h.engine.Ingest(ctx, absPath, opts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ingestion failed")
		slog.Error("ingest error", "path", absPath, "error", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"document_id": docID,
		"path":        absPath,
	})
}

// POST /query
func (h *handler) handleQuery(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	var req struct {
		Question    string  `json:"question"`
		MaxResults  int     `json:"max_results,omitempty"`
		MaxRounds   int     `json:"max_rounds,omitempty"`
		WeightVec   float64 `json:"weight_vector,omitempty"`
		WeightFTS   float64 `json:"weight_fts,omitempty"`
		WeightGraph float64 `json:"weight_graph,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Question == "" {
		writeError(w, http.StatusBadRequest, "question is required")
		return
	}

	// Bound parameters.
	if req.MaxResults < 0 || req.MaxResults > 100 {
		req.MaxResults = 0 // use default
	}
	if req.MaxRounds < 0 || req.MaxRounds > 10 {
		req.MaxRounds = 0 // use default
	}

	var opts []goreason.QueryOption
	if req.MaxResults > 0 {
		opts = append(opts, goreason.WithMaxResults(req.MaxResults))
	}
	if req.MaxRounds > 0 {
		opts = append(opts, goreason.WithMaxRounds(req.MaxRounds))
	}
	if req.WeightVec > 0 || req.WeightFTS > 0 || req.WeightGraph > 0 {
		opts = append(opts, goreason.WithWeights(req.WeightVec, req.WeightFTS, req.WeightGraph))
	}

	answer, err := h.engine.Query(ctx, req.Question, opts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		slog.Error("query error", "question", req.Question, "error", err)
		return
	}

	writeJSON(w, http.StatusOK, answer)
}

// POST /update
func (h *handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	changed, err := h.engine.Update(ctx, req.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		slog.Error("update error", "path", req.Path, "error", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":    req.Path,
		"changed": changed,
	})
}

// POST /update-all
func (h *handler) handleUpdateAll(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	results, err := h.engine.UpdateAll(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update-all failed")
		slog.Error("update-all error", "error", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

// DELETE /documents/{id}
func (h *handler) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}

	if err := h.engine.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		slog.Error("delete error", "document_id", id, "error", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GET /documents
func (h *handler) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := h.engine.ListDocuments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		slog.Error("list documents error", "error", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"documents": docs,
	})
}

// GET /health
func (h *handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf("%s", msg)})
}
