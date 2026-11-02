package opencontent

// NotifySubject is the NATS subject Open Content publishes a Notification on,
// for core to send to the user's open browser tabs over its websocket. Every
// core server subscribes, without a queue group, since each holds different
// tabs. It's plain NATS, not JetStream: like a user with no open tab, a
// message no server is listening for is dropped.
const NotifySubject = "opencontent.notify"

// Notification is one websocket message to a user.
type Notification struct {
	UserID uint `json:"user_id"`
	// Event is the message's type, such as "visits".
	Event string `json:"event"`
	// ActivityID is the activity the message is about, if any. Core ends that
	// activity when the user's tab closes.
	ActivityID int64  `json:"activity_id,omitempty"`
	Text       string `json:"text,omitempty"`
}
