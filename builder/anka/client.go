package anka

import "errors"

type Client struct {
}

func (c *Client) CreateDisk() error {
	return errors.New("Not implemented")
}
