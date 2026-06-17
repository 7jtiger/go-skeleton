package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"livein-web/model"
)

type FileStore struct {
	mu          sync.RWMutex
	contentPath string
	inquiryPath string
}

func New(contentPath, inquiryPath string) *FileStore {
	return &FileStore{
		contentPath: contentPath,
		inquiryPath: inquiryPath,
	}
}

func (s *FileStore) GetContent() (*model.Content, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.contentPath)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	var content model.Content
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("parse content: %w", err)
	}

	return &content, nil
}

func (s *FileStore) UpdateContent(content *model.Content) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}

	return writeAtomic(s.contentPath, data)
}

func (s *FileStore) ListInquiries() ([]model.Inquiry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.inquiryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.Inquiry{}, nil
		}
		return nil, fmt.Errorf("read inquiries: %w", err)
	}

	var inquiries []model.Inquiry
	if len(data) == 0 {
		return []model.Inquiry{}, nil
	}
	if err := json.Unmarshal(data, &inquiries); err != nil {
		return nil, fmt.Errorf("parse inquiries: %w", err)
	}

	return inquiries, nil
}

func (s *FileStore) AddInquiry(inquiry model.Inquiry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inquiries, err := s.readInquiriesLocked()
	if err != nil {
		return err
	}

	inquiries = append(inquiries, inquiry)

	data, err := json.MarshalIndent(inquiries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal inquiries: %w", err)
	}

	return writeAtomic(s.inquiryPath, data)
}

func (s *FileStore) readInquiriesLocked() ([]model.Inquiry, error) {
	data, err := os.ReadFile(s.inquiryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.Inquiry{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []model.Inquiry{}, nil
	}

	var inquiries []model.Inquiry
	if err := json.Unmarshal(data, &inquiries); err != nil {
		return nil, err
	}
	return inquiries, nil
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}
