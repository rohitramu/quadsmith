package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type blobResponse struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	MIMEType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
	IsDuplicate bool   `json:"is_duplicate,omitempty"`
}

func initBlobCommands(rootCmd *cobra.Command, targetURL string, httpClient *http.Client) {
	blobCmd := &cobra.Command{
		Use:   "blob",
		Short: "Upload and manage binary blobs (images, 3D models, videos)",
	}

	uploadCmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload a file (image, 3D STL/STEP/3MF model, or video)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			stat, err := file.Stat()
			if err != nil {
				return err
			}

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", filepath.Base(filePath))
			if err != nil {
				return err
			}
			if _, err := io.Copy(part, file); err != nil {
				return err
			}
			if err := writer.Close(); err != nil {
				return err
			}

			endpoint := strings.TrimRight(targetURL, "/") + "/api/blobs"
			req, err := http.NewRequest(http.MethodPost, endpoint, &body)
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("upload request failed: %w", err)
			}
			defer resp.Body.Close()

			respBytes, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
				var errMap map[string]string
				if json.Unmarshal(respBytes, &errMap) == nil && errMap["error"] != "" {
					return fmt.Errorf("upload failed (%d): %s", resp.StatusCode, errMap["error"])
				}
				return fmt.Errorf("upload failed with HTTP status %d: %s", resp.StatusCode, string(respBytes))
			}

			var res blobResponse
			if err := json.Unmarshal(respBytes, &res); err != nil {
				return fmt.Errorf("failed parsing server response: %w", err)
			}

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "✓ File uploaded successfully!")
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID:\t%s\n", res.ID)
			fmt.Fprintf(w, "URL:\t%s\n", res.URL)
			fmt.Fprintf(w, "Filename:\t%s\n", res.Filename)
			fmt.Fprintf(w, "MIME Type:\t%s\n", res.MIMEType)
			fmt.Fprintf(w, "Size:\t%s (%d bytes)\n", formatFileSize(stat.Size()), stat.Size())
			return w.Flush()
		},
	}
	blobCmd.AddCommand(uploadCmd)

	var outputFile string
	var metadataOnly bool

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Download or inspect a blob by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			baseURL := strings.TrimRight(targetURL, "/")

			if metadataOnly {
				req, err := http.NewRequest(http.MethodGet, baseURL+"/api/blobs/"+id+"/metadata", nil)
				if err != nil {
					return err
				}
				resp, err := httpClient.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("failed retrieving metadata (status %d)", resp.StatusCode)
				}
				var res blobResponse
				if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
					return err
				}
				if jsonOut {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(res)
				}
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				fmt.Fprintf(w, "ID:\t%s\n", res.ID)
				fmt.Fprintf(w, "URL:\t%s\n", res.URL)
				fmt.Fprintf(w, "Filename:\t%s\n", res.Filename)
				fmt.Fprintf(w, "MIME Type:\t%s\n", res.MIMEType)
				fmt.Fprintf(w, "Size:\t%s (%d bytes)\n", formatFileSize(res.SizeBytes), res.SizeBytes)
				return w.Flush()
			}

			req, err := http.NewRequest(http.MethodGet, baseURL+"/blobs/"+id, nil)
			if err != nil {
				return err
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
				return fmt.Errorf("download failed (status %d)", resp.StatusCode)
			}

			outPath := outputFile
			if outPath == "" {
				// Extract filename from header or use ID
				disp := resp.Header.Get("Content-Disposition")
				if strings.Contains(disp, "filename=") {
					parts := strings.Split(disp, "filename=")
					if len(parts) > 1 {
						outPath = strings.Trim(parts[1], `"'`)
					}
				}
				if outPath == "" {
					outPath = id
				}
			}

			outFile, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("failed creating output file %s: %w", outPath, err)
			}
			defer outFile.Close()

			n, err := io.Copy(outFile, resp.Body)
			if err != nil {
				return fmt.Errorf("failed writing file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Downloaded %s (%d bytes) to %s\n", id, n, outPath)
			return nil
		},
	}
	getCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Destination file path for download")
	getCmd.Flags().BoolVarP(&metadataOnly, "metadata", "m", false, "Retrieve metadata only without downloading bytes")
	blobCmd.AddCommand(getCmd)

	rootCmd.AddCommand(blobCmd)
}

func formatFileSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
