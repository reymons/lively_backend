package message

type Subscribe struct {
	Topic string `json:"topic"`
}

type Unsubscribe struct {
	Topic string `json:"topic"`
}
