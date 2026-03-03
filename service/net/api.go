package net

import (
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	serverAddr = "https://typstify.com"
	apiKey     = "gX1YrqioTqKK5dksTfKm3iMbQpvecbqrWo8kxLXt01TFw029samsvCgr4Ge6LlT9"
)

type Remote struct {
	client *http.Client
	server string
	apiKey string
}

func NewRemote() *Remote {
	if _, err := url.ParseRequestURI(serverAddr); err != nil {
		panic(err)
	}

	return &Remote{
		client: &http.Client{
			Timeout: time.Second * 60,
		},
		server: serverAddr,
		apiKey: apiKey,
	}
}

func (r *Remote) invoker(path string) *HttpInvoker {
	var addr string
	if uri, err := url.ParseRequestURI(path); err == nil && uri.Scheme != "" {
		// is an absolute path
		addr = path
	} else {
		var err error
		addr, err = url.JoinPath(r.server, path)
		if err != nil {
			return nil
		}
	}
	log.Println("requesting: ", addr)
	invoker := NewInvoker(r.client, addr)
	invoker.SetAuthorization(r.apiKey)
	return invoker
}

func (r *Remote) RegisterDevice(req *DeviceInfo) error {
	const path = "/api/devices/register"
	invoker := r.invoker(path)
	err := invoker.Post(req).Decode(nil)
	if err != nil {
		log.Println("register device failed: ", err)
		return err
	}
	return nil
}

func (r *Remote) CheckUpdate(req *UpdateCheckReq) (*ReleaseInfo, error) {
	const path = "/api/devices/check-update"
	invoker := r.invoker(path)
	var resp ReleaseInfo

	err := invoker.Post(req).Decode(&resp)
	if err != nil {
		log.Println("read from network failed: ", err)
		return nil, err
	}

	return &resp, nil
}
