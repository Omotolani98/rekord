package live

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const dialTimeout = 2 * time.Second

func Do(path string, req Request) (Response, error) {
	conn, err := net.DialTimeout("unix", path, dialTimeout)
	if err != nil {
		return Response{}, err
	}
	defer func() { _ = conn.Close() }()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, fmt.Errorf("send request: %w", err)
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("read response: %w", err)
	}
	if !resp.OK && resp.Error != "" {
		return resp, fmt.Errorf("%s", resp.Error)
	}
	return resp, nil
}

func Ping(path string) bool {
	_, err := Do(path, Request{Op: "status"})
	return err == nil
}
