// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"golang.org/x/exp/jsonrpc2"
	"golang.org/x/telemetry/crashmonitor"
)

var (
	// RequestCancelledError should be used when a request is cancelled early.
	RequestCancelledError = jsonrpc2.NewError(-32800, "JSON RPC cancelled")
)

type ClientCloser interface {
	Client
	io.Closer
}

type connSender interface {
	io.Closer

	Notify(ctx context.Context, method string, params interface{}) error
	Call(ctx context.Context, method string, params interface{}) *jsonrpc2.AsyncCall
}

type clientDispatcher struct {
	sender connSender
}

func (c *clientDispatcher) Close() error {
	return c.sender.Close()
}

// ClientDispatcher returns a Client that dispatches LSP requests across the
// given jsonrpc2 connection.
func ClientDispatcher(conn *jsonrpc2.Connection) ClientCloser {
	return &clientDispatcher{sender: conn}
}

// ServerDispatcher returns a Server that dispatches LSP requests across the
// given jsonrpc2 connection.
func ServerDispatcher(conn *jsonrpc2.Connection) Server {
	return &serverDispatcher{sender: conn}
}

/*
func ServerDispatcherV2(conn *jsonrpc2_v2.Connection) Server {
	return &serverDispatcher{sender: clientConnV2{conn}}
}
*/

type serverDispatcher struct {
	sender connSender
}

func ClientHandler(client Client, handler jsonrpc2.Handler) jsonrpc2.Handler {
	return jsonrpc2.HandlerFunc(func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		if ctx.Err() != nil {
			return nil, RequestCancelledError
		}
		handled, resp, err := clientDispatch(ctx, client, req)
		if handled || err != nil {
			return resp, err
		}
		return handler.Handle(ctx, req)
	})
}

func ServerHandler(server Server, handler jsonrpc2.Handler) jsonrpc2.Handler {
	return jsonrpc2.HandlerFunc(func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		if ctx.Err() != nil {
			return nil, RequestCancelledError
		}
		handled, resp, err := serverDispatch(ctx, server, req)
		if handled || err != nil {
			return resp, err
		}
		return handler.Handle(ctx, req)
	})
}

func Call(ctx context.Context, conn jsonrpc2.Connection, method string, params interface{}, result interface{}) error {
	call := conn.Call(ctx, method, params)
	if ctx.Err() != nil {
		conn.Cancel(call.ID())
	}
	return call.Await(ctx, result)
}

func cancelCall(ctx context.Context, sender connSender, id jsonrpc2.ID) {
	ctx = Detach(ctx)
	// Note that only *jsonrpc2.ID implements json.Marshaler.
	sender.Notify(ctx, "$/cancelRequest", &CancelParams{ID: &id})
}

// UnmarshalJSON unmarshals msg into the variable pointed to by
// params. In JSONRPC, optional messages may be
// "null", in which case it is a no-op.
func UnmarshalJSON(msg json.RawMessage, v any) error {
	if len(msg) == 0 || bytes.Equal(msg, []byte("null")) {
		return nil
	}
	return json.Unmarshal(msg, v)
}

func sendParseError(ctx context.Context, err error) error {
	return fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
}

// NonNilSlice returns x, or an empty slice if x was nil.
//
// (Many slice fields of protocol structs must be non-nil
// to avoid being encoded as JSON "null".)
func NonNilSlice[T comparable](x []T) []T {
	if x == nil {
		return []T{}
	}
	return x
}

func recoverHandlerPanic(method string) {
	// Report panics in the handler goroutine,
	// unless we have enabled the monitor,
	// which reports all crashes.
	if !crashmonitor.Supported() {
		defer func() {
			if x := recover(); x != nil {
				slog.Debug("panic in request", slog.String("method", method))
				panic(x)
			}
		}()
	}
}

// Detach returns a context that keeps all the values of its parent context
// but detaches from the cancellation and error handling.
func Detach(ctx context.Context) context.Context { return detachedContext{ctx} }

type detachedContext struct{ parent context.Context }

func (v detachedContext) Deadline() (time.Time, bool)       { return time.Time{}, false }
func (v detachedContext) Done() <-chan struct{}             { return nil }
func (v detachedContext) Err() error                        { return nil }
func (v detachedContext) Value(key interface{}) interface{} { return v.parent.Value(key) }
