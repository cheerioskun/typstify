package lsp

import (
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"looz.ws/typstify/lsp/protocol"
)

type DocReader func() string

type document struct {
	// Every open file that has been sent to LSP server has a version,
	// it is bumped when it is updated and sent again.
	Version      int
	Path         string
	URI          protocol.DocumentURI
	Removed      bool
	updateTime   time.Time
	lastSyncTime time.Time
	reader       io.ReadSeeker
	mu           sync.Mutex
	readerMu     sync.Mutex
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

func (doc *document) Synced() bool {
	doc.mu.Lock()
	defer doc.mu.Unlock()
	return doc.updateTime.Equal(doc.lastSyncTime)
}

func (doc *document) IsNew() bool {
	doc.mu.Lock()
	defer doc.mu.Unlock()
	return doc.Version == 0
}

func (doc *document) MakrSynced() {
	doc.mu.Lock()
	defer doc.mu.Unlock()
	if doc.updateTime.After(doc.lastSyncTime) {
		doc.lastSyncTime = doc.updateTime
	}
}

func newDocumentCache() *documentCache {
	return &documentCache{
		// recentUpdated: make(map[string]time.Time),
		docs: make(map[string]*document),
	}
}

func (c *documentCache) add(filePath string, reader io.ReadSeeker) {
	doc := &document{
		Version:    0,
		Path:       filePath,
		URI:        protocol.URIFromPath(filePath),
		reader:     reader,
		Removed:    false,
		updateTime: time.Now(),
	}
	c.docs[filePath] = doc
	log.Println("Added document", filePath)
}

func (c *documentCache) Update(filePath string, reader io.ReadSeeker) {
	c.mu.Lock()
	defer c.mu.Unlock()

	doc, exists := c.docs[filePath]
	if !exists {
		c.add(filePath, reader)
		return
	}

	if doc.Synced() {
		doc.updateTime = time.Now()
		// doc.reader = reader
		doc.Version++
	} else {
		doc.updateTime = time.Now()
	}
}

func (c *documentCache) Remove(filePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if doc, exists := c.docs[filePath]; exists {
		doc.Removed = true
		doc.updateTime = time.Now()
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
