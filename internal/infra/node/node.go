package node

import (
	"sync"

	"github.com/google/uuid"
)

var (
	once   sync.Once
	nodeID string
)

func ID() string {
	once.Do(func() {
		nodeID = uuid.Must(uuid.NewV7()).String()
	})
	return nodeID
}
