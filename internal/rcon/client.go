package rcon

import (
	"context"
	"fmt"

	gorcon "github.com/gorcon/rcon"
)

type Conn interface {
	Execute(cmd string) (string, error)
	Close()
}

type Client struct {
	conn Conn
}

func NewClientWithConn(conn Conn) *Client {
	return &Client{conn: conn}
}

func Dial(host string, port int, password string) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := gorcon.Dial(addr, password)
	if err != nil {
		return nil, fmt.Errorf("rcon dial %s: %w", addr, err)
	}
	return &Client{conn: &gorconConn{conn}}, nil
}

func (c *Client) Execute(cmd string) (string, error) {
	resp, err := c.conn.Execute(cmd)
	if err != nil {
		return "", fmt.Errorf("execute %q: %w", cmd, err)
	}
	return resp, nil
}

func (c *Client) send(ctx context.Context, cmd string) (string, error) {
	return c.Execute(cmd)
}

func (c *Client) PrepareBackup(ctx context.Context) error {
	if _, err := c.send(ctx, "save-off"); err != nil {
		return err
	}
	if _, err := c.send(ctx, "save-all"); err != nil {
		return err
	}
	return nil
}

func (c *Client) RestoreSave(ctx context.Context) error {
	_, err := c.send(ctx, "save-on")
	return err
}

func (c *Client) Close() {
	c.conn.Close()
}

type gorconConn struct {
	conn *gorcon.Conn
}

func (g *gorconConn) Execute(cmd string) (string, error) {
	return g.conn.Execute(cmd)
}

func (g *gorconConn) Close() {
	_ = g.conn.Close()
}
