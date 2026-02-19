package bus

import "github.com/nats-io/nats.go"

const (
	SubjectMatchCreated = "events.match.created"
	SubjectSessionStart = "events.session.started"
	SubjectPlayerLogin  = "events.player.logged_in"
)

// Connect creates a NATS connection for message bus communication.
func Connect(url string) (*nats.Conn, error) {
	return nats.Connect(url)
}
