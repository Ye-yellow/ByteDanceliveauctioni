package realtime

type MessageType string

const (
	MessageLotUpdated  MessageType = "lot.updated"
	MessageBidAccepted MessageType = "bid.accepted"
	MessageLotSettled  MessageType = "lot.settled"
	MessageError       MessageType = "error"
)

type Envelope struct {
	Type MessageType `json:"type"`
	Data interface{} `json:"data"`
}
