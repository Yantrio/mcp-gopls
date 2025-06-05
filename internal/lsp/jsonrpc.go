package lsp

import (
	"context"
	"encoding/json"
	"io"
	"os/exec"

	"github.com/sourcegraph/jsonrpc2"
)

type readWriteCloser struct {
	io.ReadCloser
	io.WriteCloser
}

func (rwc readWriteCloser) Close() error {
	err1 := rwc.ReadCloser.Close()
	err2 := rwc.WriteCloser.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func newProcessConnection(cmd *exec.Cmd) (*jsonrpc2.Conn, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	stream := jsonrpc2.NewBufferedStream(
		readWriteCloser{stdout, stdin},
		jsonrpc2.VSCodeObjectCodec{},
	)

	handler := &serverHandler{}
	conn := jsonrpc2.NewConn(
		context.Background(),
		stream,
		handler,
	)

	return conn, nil
}

type serverHandler struct {
	diagnostics map[string][]Diagnostic
}

func (h *serverHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	switch req.Method {
	case "textDocument/publishDiagnostics":
		var params PublishDiagnosticsParams
		if req.Params != nil && json.Unmarshal(*req.Params, &params) == nil {
			if h.diagnostics == nil {
				h.diagnostics = make(map[string][]Diagnostic)
			}
			h.diagnostics[params.URI] = params.Diagnostics
		}
	case "window/logMessage":
		// Ignore log messages for now
	case "$/progress":
		// Ignore progress notifications
	case "window/showMessage":
		// Ignore show message notifications
	default:
		// Unknown notification, ignore
	}
}
