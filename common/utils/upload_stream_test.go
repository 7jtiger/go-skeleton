package utils

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type uploadTransport func(*http.Request) (*http.Response, error)

func (f uploadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUploadStreamsMultipart(t *testing.T) {
	var source bytes.Buffer
	w := multipart.NewWriter(&source)
	part, _ := w.CreateFormFile("files", "test.jpg")
	payload := strings.Repeat("image", 20000)
	_, _ = io.WriteString(part, payload)
	_ = w.Close()
	req := httptest.NewRequest("POST", "/", &source)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(1024); err != nil {
		t.Fatal(err)
	}
	defer req.MultipartForm.RemoveAll()
	previous := cloudflareUploadClient
	defer func() { cloudflareUploadClient = previous }()
	cloudflareUploadClient = &http.Client{Transport: uploadTransport(func(r *http.Request) (*http.Response, error) {
		if r.ContentLength > 0 {
			t.Fatal("upload buffered instead of streamed")
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		part, err := reader.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(part)
		if err != nil || string(data) != payload || part.FileName() != "test.jpg" {
			t.Fatal("incorrect multipart payload")
		}
		if _, err := reader.NextPart(); err != io.EOF {
			t.Fatalf("multipart trailer: %v", err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})}
	if _, status, err := postCldflrMultipart("https://upload.invalid", req.MultipartForm.File["files"][0], "test"); err != nil || status != 200 {
		t.Fatalf("status=%d err=%v", status, err)
	}
	cloudflareUploadClient.Transport = uploadTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 413, Body: io.NopCloser(strings.NewReader("too large")), Header: make(http.Header)}, nil
	})
	if _, status, err := postCldflrMultipart("https://upload.invalid", req.MultipartForm.File["files"][0], "test"); err != nil || status != 413 {
		t.Fatalf("early rejection: status=%d err=%v", status, err)
	}
}
