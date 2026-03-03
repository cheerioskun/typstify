package lsp

import (
	"context"
	"log"
	"strings"

	"looz.ws/typstify/service/settings"
	"looz.ws/typstify/utils"
)

var (
	// Use a singleton to serve a workspace.
	lspClient *Client
	version   string
)

// use init function to setup PATH.
func Init(externalDir string) {
	utils.LookupExecutable(lspServerName, externalDir)
}

func GetLspClient(workspace string, setting *settings.Settings) *Client {
	if lspClient != nil && lspClient.server.Workspace() == workspace {
		return lspClient
	}

	if lspClient != nil {
		lspClient.Stop()
	}

	lspClient = newClient(newServer(workspace))
	err := lspClient.Start(context.Background(), setting)
	if err != nil {
		log.Println("LSP client setup failed: ", err)
		return nil
	}

	log.Println("LSP client is setup")
	return lspClient
}

func StopLsp() {
	if lspClient != nil {
		lspClient.Stop()
	}
}

// Version returns the LSP server version.
func Version() string {
	if version == "" {
		cmd := newCmd(context.Background(), "-V")
		out, _ := cmd.Output()
		version = strings.TrimSpace(string(out))
	}

	return version
}
