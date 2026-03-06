package client

type Client struct {
	ServerURL string
}

func NewClient() *Client {
	return &Client{ServerURL: "http://0.0.0.0:5050"}
}
