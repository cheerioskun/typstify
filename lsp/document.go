package lsp

import (
	"errors"
	"io"
	"sync"

	"looz.ws/typstify/lsp/protocol"
)

type DocReader func() string

type document struct {
	// Every open file that has been sent to LSP server has a version,
	// it is bumped when it is updated and sent again.
	Version         int
	Path            string
	URI             protocol.DocumentURI
	Removed         bool
	lastSyncVersion int
	isOpenSynced    bool
	reader          io.ReadSeeker
	mu              sync.Mutex
	readerMu        sync.Mutex
}

// documentCache holds cached data of the document.
type documentCache struct {
	docs map[string]*document
	mu   sync.Mutex
}

func (doc *document) Content() string {
	doc.readerMu.Lock()
	defer doc.readerMu.Unlock()
	if doc.reader != nil {
		doc.reader.Seek(0, io.SeekStart)
		c, err := io.ReadAll(doc.reader)
		if err != nil {
			return ""
		}

		return string(c)
	}

	return ""
}

func (doc *document) IsNew() bool {
	doc.mu.Lock()
	defer doc.mu.Unlock()
	return doc.Version == 0 || !doc.isOpenSynced
}

// Synced should check if the Last Sent Version matches the Current Version
func (doc *document) NeedsSync() bool {
	doc.mu.Lock()
	defer doc.mu.Unlock()
	return !doc.isOpenSynced || doc.lastSyncVersion < doc.Version
}

func (doc *document) MarkSynced(version int) {
	doc.mu.Lock()
	defer doc.mu.Unlock()
	if version > doc.lastSyncVersion {
		doc.lastSyncVersion = version
	}

	doc.isOpenSynced = true
}

func newDocumentCache() *documentCache {
	return &documentCache{
		// recentUpdated: make(map[string]time.Time),
		docs: make(map[string]*document),
	}
}

func (c *documentCache) add(filePath string, reader io.ReadSeeker) {
	doc := &document{
		Version: 0,
		Path:    filePath,
		URI:     protocol.URIFromPath(filePath),
		reader:  reader,
		Removed: false,
	}
	c.docs[filePath] = doc
}

func (c *documentCache) Update(filePath string, reader io.ReadSeeker) {
	c.mu.Lock()
	defer c.mu.Unlock()

	doc, exists := c.docs[filePath]
	if !exists {
		c.add(filePath, reader)
		return
	}

	doc.mu.Lock()
	// Always increment version for LSP compliance
	doc.Version++
	doc.reader = reader
	doc.mu.Unlock()
}

func (c *documentCache) Remove(filePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if doc, exists := c.docs[filePath]; exists {
		doc.Removed = true
	}
}

// Get returns a Document that has not synchronized with LSP server since the
// last update time. If there is not update, it returns nil.
func (c *documentCache) Get(filePath string) (*document, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	doc, exists := c.docs[filePath]
	if !exists {
		return nil, errors.New("document does not exist")
	}

	return doc, nil
}
