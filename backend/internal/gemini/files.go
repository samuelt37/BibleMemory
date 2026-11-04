package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const filesBase = "https://generativelanguage.googleapis.com"

type File struct {
	Name     string `json:"name"`
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	State    string `json:"state"`
}

func (c *Client) UploadFile(ctx context.Context, r io.Reader, size int64, mimeType, displayName string) (*File, error) {
	apiKey, err := c.getAPIKey()
	if err != nil {
		return nil, err
	}

	meta, _ := json.Marshal(map[string]any{"file": map[string]string{"display_name": displayName}})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, filesBase+"/upload/v1beta/files", bytes.NewReader(meta))
	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("X-Goog-Upload-Protocol", "resumable")
	req.Header.Set("X-Goog-Upload-Command", "start")
	req.Header.Set("X-Goog-Upload-Header-Content-Length", strconv.FormatInt(size, 10))
	req.Header.Set("X-Goog-Upload-Header-Content-Type", mimeType)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	uploadURL := resp.Header.Get("X-Goog-Upload-URL")
	if uploadURL == "" {
		return nil, fmt.Errorf("files api: no upload url (status %d: %s)", resp.StatusCode, string(bodyBytes))
	}

	req, _ = http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, r)
	req.ContentLength = size
	req.Header.Set("X-Goog-Upload-Offset", "0")
	req.Header.Set("X-Goog-Upload-Command", "upload, finalize")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("files api upload: %d %s", resp.StatusCode, b)
	}
	var out struct {
		File File `json:"file"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	f := &out.File
	for f.State == "PROCESSING" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}
		if f, err = c.getFile(ctx, apiKey, f.Name); err != nil {
			return nil, err
		}
	}
	if f.State != "ACTIVE" {
		return nil, fmt.Errorf("files api: file ended in state %s", f.State)
	}
	return f, nil
}

func (c *Client) getFile(ctx context.Context, apiKey, name string) (*File, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, filesBase+"/v1beta/"+name, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var f File
	if err := json.NewDecoder(resp.Body).Decode(&f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (c *Client) DeleteFile(ctx context.Context, name string) error {
	apiKey, err := c.getAPIKey()
	if err != nil {
		return err
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, filesBase+"/v1beta/"+name, nil)
	req.Header.Set("x-goog-api-key", apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
