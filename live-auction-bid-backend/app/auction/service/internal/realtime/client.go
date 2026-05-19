package realtime

type Client interface {
	SendJSON(v interface{}) error
}
