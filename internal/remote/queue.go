package remote

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/user/rt/internal/config"
)

type QueueItem struct {
	Type     string      `json:"type"`
	Evidence *EvidenceReq `json:"evidence,omitempty"`
	Finding  *FindingReq  `json:"finding,omitempty"`
	Cred     *CredReq     `json:"cred,omitempty"`
}

var queueMu sync.Mutex

func queuePath() string {
	return filepath.Join(config.Home(), ".sync-queue.json")
}

func Enqueue(item QueueItem) error {
	queueMu.Lock()
	defer queueMu.Unlock()

	items, _ := readQueue()
	items = append(items, item)
	return writeQueue(items)
}

func QueueLen() int {
	queueMu.Lock()
	defer queueMu.Unlock()
	items, _ := readQueue()
	return len(items)
}

func DrainQueue(client *Client) (sent, failed int) {
	queueMu.Lock()
	defer queueMu.Unlock()

	items, _ := readQueue()
	if len(items) == 0 {
		return 0, 0
	}

	var remaining []QueueItem
	for _, item := range items {
		var err error
		switch item.Type {
		case "evidence":
			if item.Evidence != nil {
				_, err = client.SubmitEvidence(*item.Evidence)
			}
		case "finding":
			if item.Finding != nil {
				_, err = client.SubmitFinding(*item.Finding)
			}
		case "cred":
			if item.Cred != nil {
				err = client.SubmitCredential(*item.Cred)
			}
		}
		if err != nil {
			remaining = append(remaining, item)
			failed++
		} else {
			sent++
		}
	}

	if len(remaining) == 0 {
		os.Remove(queuePath())
	} else {
		writeQueue(remaining)
	}
	return sent, failed
}

func readQueue() ([]QueueItem, error) {
	data, err := os.ReadFile(queuePath())
	if err != nil {
		return nil, err
	}
	var items []QueueItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func writeQueue(items []QueueItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return os.WriteFile(queuePath(), data, 0600)
}
