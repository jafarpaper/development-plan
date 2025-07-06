package valueobject

import (
	"crypto/rand"
	"fmt"
	"strings"
)

type ActivityLogID string

func NewActivityLogID() ActivityLogID {
	return ActivityLogID(generateID())
}

func (id ActivityLogID) String() string {
	return string(id)
}

func (id ActivityLogID) IsValid() bool {
	return len(strings.TrimSpace(string(id))) > 0
}

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}
