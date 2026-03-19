package ingestor

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ingestor connects to the F1 live timing stream, maintains an in-memory
// snapshot of the current state per topic, and flushes changes to PostgreSQL.
type Ingestor struct {
	db    *pgxpool.Pool
	mu    sync.RWMutex
	state map[string]map[string]any // topic → merged state
}

func New(db *pgxpool.Pool) *Ingestor {
	return &Ingestor{
		db:    db,
		state: make(map[string]map[string]any),
	}
}

// Run connects and keeps reconnecting with exponential backoff.
// Call in a goroutine; respects ctx cancellation.
func (ing *Ingestor) Run(ctx context.Context) {
	backoff := 2 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Println("ingestor: connecting to F1 live timing...")
		client := NewClient(func(topic string, data json.RawMessage) {
			ing.handle(topic, data)
		})

		if err := client.Connect(); err != nil {
			log.Printf("ingestor: connect error: %v — retrying in %s", err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				backoff = min(backoff*2, 60*time.Second)
				continue
			}
		}

		backoff = 2 * time.Second
		log.Println("ingestor: connected")

		if err := client.ReadLoop(); err != nil {
			log.Printf("ingestor: read loop ended: %v", err)
		}
		client.Close()
	}
}

// Snapshot returns a deep copy of the current state for a topic.
func (ing *Ingestor) Snapshot(topic string) map[string]any {
	ing.mu.RLock()
	defer ing.mu.RUnlock()
	s, ok := ing.state[topic]
	if !ok {
		return nil
	}
	// shallow copy is fine for read-only API responses
	out := make(map[string]any, len(s))
	for k, v := range s {
		out[k] = v
	}
	return out
}

// handle is called for each incoming feed message.
func (ing *Ingestor) handle(topic string, raw json.RawMessage) {
	// Decompress .z topics (CarData.z, Position.z).
	if isCompressed(topic) {
		decoded, err := decompress(raw)
		if err != nil {
			log.Printf("ingestor: decompress %s: %v", topic, err)
			return
		}
		raw = decoded
	}

	delta, err := unmarshalMap(raw)
	if err != nil {
		// Some messages are scalars (e.g. LapCount is just a number) —
		// wrap them so we can store uniformly.
		delta = map[string]any{"value": json.RawMessage(raw)}
	}

	ing.mu.Lock()
	existing := ing.state[topic]
	if existing == nil {
		ing.state[topic] = delta
	} else {
		ing.state[topic] = deepMerge(existing, delta)
	}
	merged := ing.state[topic]
	ing.mu.Unlock()

	ing.persist(topic, merged)
}

// persist upserts the current state of a topic into the DB.
func (ing *Ingestor) persist(topic string, state map[string]any) {
	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = ing.db.Exec(ctx, `
		INSERT INTO live_state (topic, data, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (topic) DO UPDATE
		  SET data = $2, updated_at = NOW()
	`, topic, data)
	if err != nil {
		log.Printf("ingestor: db write %s: %v", topic, err)
	}
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
