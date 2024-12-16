package blacklist

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type BlacklistEntry struct {
	IP        string    `json:"ip"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Blacklist struct {
	Entries []BlacklistEntry `json:"entries"`
	mu      sync.RWMutex
	file    string
}

func NewBlacklist(file string) (*Blacklist, error) {
	bl := &Blacklist{
		Entries: []BlacklistEntry{},
		file:    file,
	}
	err := bl.Load()
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, create it with an empty blacklist
			return bl, bl.Save()
		}
		return nil, err
	}
	return bl, nil
}

func (bl *Blacklist) Load() error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	data, err := os.ReadFile(bl.file)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, bl)
}

func (bl *Blacklist) Save() error {
	data, err := json.MarshalIndent(bl, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(bl.file, data, 0644)
}

func (bl *Blacklist) Add(ip string, duration time.Duration) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	bl.mu.Lock()
	defer bl.mu.Unlock()

	expiresAt := time.Time{}
	if duration > 0 {
		expiresAt = time.Now().Add(duration)
	}

	for i, entry := range bl.Entries {
		if entry.IP == ip {
			bl.Entries[i].ExpiresAt = expiresAt
			return bl.Save()
		}
	}

	bl.Entries = append(bl.Entries, BlacklistEntry{IP: ip, ExpiresAt: expiresAt})
	return bl.Save()
}

func (bl *Blacklist) Remove(ip string) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	for i, entry := range bl.Entries {
		if entry.IP == ip {
			bl.Entries = append(bl.Entries[:i], bl.Entries[i+1:]...)
			return bl.Save()
		}
	}

	return nil // IP not found in the list
}

func (bl *Blacklist) Contains(ip string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	now := time.Now()
	for _, entry := range bl.Entries {
		if entry.IP == ip {
			if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(now) {
				return true
			}
		}
	}
	return false
}

func (bl *Blacklist) GetIPs() []string {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	var ips []string
	now := time.Now()
	for _, entry := range bl.Entries {
		if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(now) {
			ips = append(ips, entry.IP)
		}
	}
	return ips
}

func (bl *Blacklist) Cleanup() {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	now := time.Now()
	newEntries := make([]BlacklistEntry, 0, len(bl.Entries))
	for _, entry := range bl.Entries {
		if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(now) {
			newEntries = append(newEntries, entry)
		}
	}

	if len(newEntries) < len(bl.Entries) {
		bl.Entries = newEntries
		_ = bl.Save() // Ignore error as this is a background operation
	}
}

