package rcon

import (
	"context"
	"fmt"

	"github.com/gustavogebhardt/mc-backup/internal/rcon"
)

type Dialer interface {
	Dial() (rcon.Conn, error)
}

type RCONNotifier struct {
	dialer Dialer
}

func New(dialer Dialer) *RCONNotifier {
	return &RCONNotifier{dialer: dialer}
}

func (n *RCONNotifier) Notify(ctx context.Context, msg string) error {
	conn, err := n.dialer.Dial()
	if err != nil {
		return fmt.Errorf("rcon dial: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Execute("say " + msg); err != nil {
		return fmt.Errorf("say: %w", err)
	}
	return nil
}

type OSDialer struct {
	host     string
	port     int
	password string
}

func NewOSDialer(host string, port int, password string) *OSDialer {
	return &OSDialer{host: host, port: port, password: password}
}

func (d *OSDialer) Dial() (rcon.Conn, error) {
	client, err := rcon.Dial(d.host, d.port, d.password)
	if err != nil {
		return nil, err
	}
	return client, nil
}
