package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var aliases *AliasStore

// AliasStore persists IP-to-name mappings.
type AliasStore struct {
	mu    sync.RWMutex
	path  string
	Alias map[string]string `json:"alias"`
}

func newAliasStore(path string) (*AliasStore, error) {
	s := &AliasStore{
		path:  path,
		Alias: make(map[string]string),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("解析别名文件失败 %s: %w", path, err)
	}
	return s, nil
}

func (a *AliasStore) lookup(ip string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Alias[ip]
}

func (a *AliasStore) formatIP(ip string) string {
	if a == nil {
		return ip
	}
	if name := a.lookup(ip); name != "" {
		return fmt.Sprintf("%s(%s)", name, ip)
	}
	return ip
}

func (a *AliasStore) set(ip, name string) error {
	a.mu.Lock()
	a.Alias[ip] = name
	a.mu.Unlock()
	return a.save()
}

func (a *AliasStore) del(ip string) error {
	a.mu.Lock()
	delete(a.Alias, ip)
	a.mu.Unlock()
	return a.save()
}

func (a *AliasStore) list() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	m := make(map[string]string, len(a.Alias))
	for k, v := range a.Alias {
		m[k] = v
	}
	return m
}

func (a *AliasStore) save() error {
	a.mu.RLock()
	data, err := json.MarshalIndent(a, "", "  ")
	a.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("序列化别名失败: %w", err)
	}
	if err := os.WriteFile(a.path, data, 0644); err != nil {
		return fmt.Errorf("写入别名文件失败 %s: %w", a.path, err)
	}
	return nil
}
