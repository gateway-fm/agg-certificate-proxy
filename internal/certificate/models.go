package certificate

import (
	"database/sql"
	"time"
)

// Certificate represents a stored certificate
type Certificate struct {
	ID          int64
	RawProto    []byte
	ReceivedAt  time.Time
	ProcessedAt sql.NullTime
	Metadata    string
}
