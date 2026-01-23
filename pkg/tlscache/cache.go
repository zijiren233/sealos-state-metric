// Package tlscache provides TLS certificate caching with automatic reloading using fsnotify
package tlscache

import (
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

// Cache caches TLS certificate with fsnotify-based reloading
type Cache struct {
	mu       sync.RWMutex
	cert     *tls.Certificate
	certFile string
	keyFile  string
	watcher  *fsnotify.Watcher
	stopCh   chan struct{}
}

// New creates a new certificate cache with file watching
func New(certFile, keyFile string) (*Cache, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	c := &Cache{
		certFile: certFile,
		keyFile:  keyFile,
		watcher:  watcher,
		stopCh:   make(chan struct{}),
	}

	// Load initial certificate
	if err := c.loadCertificate(); err != nil {
		watcher.Close()
		return nil, err
	}

	// Start watching files
	if err := c.startWatching(); err != nil {
		watcher.Close()
		return nil, err
	}

	return c, nil
}

// startWatching starts watching certificate files for changes
func (c *Cache) startWatching() error {
	if err := c.watcher.Add(c.certFile); err != nil {
		return fmt.Errorf("failed to watch cert file: %w", err)
	}

	if err := c.watcher.Add(c.keyFile); err != nil {
		return fmt.Errorf("failed to watch key file: %w", err)
	}

	// Start watch loop in goroutine
	go c.watchLoop()

	// Start polling loop as backup (in case fsnotify misses events)
	go c.pollLoop()

	log.WithFields(log.Fields{
		"certFile": c.certFile,
		"keyFile":  c.keyFile,
	}).Info("Certificate auto-reload enabled via fsnotify")

	return nil
}

// watchLoop monitors file changes using fsnotify
func (c *Cache) watchLoop() {
	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				return
			}

			// Reload on Write, Create, Chmod, or Remove events
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Chmod|fsnotify.Remove) != 0 {
				log.WithFields(log.Fields{
					"file": event.Name,
					"op":   event.Op.String(),
				}).Debug("Certificate file changed, reloading")

				if err := c.loadCertificate(); err != nil {
					log.WithError(err).Error("Failed to reload certificate after file change")
				} else {
					log.Info("Certificate reloaded successfully")
				}
			}

		case err, ok := <-c.watcher.Errors:
			if !ok {
				return
			}

			log.WithError(err).Error("Certificate watcher error")

		case <-c.stopCh:
			return
		}
	}
}

// pollLoop periodically checks for certificate changes as backup
func (c *Cache) pollLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Just trigger a reload attempt - loadCertificate is idempotent
			_ = c.loadCertificate()

		case <-c.stopCh:
			return
		}
	}
}

// loadCertificate loads the certificate from disk
func (c *Cache) loadCertificate() error {
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	c.mu.Lock()
	c.cert = &cert
	c.mu.Unlock()

	log.Debug("Certificate loaded successfully")

	return nil
}

// GetCertificate returns the cached certificate (compatible with tls.Config.GetCertificate)
func (c *Cache) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cert == nil {
		return nil, errors.New("no certificate loaded")
	}

	return c.cert, nil
}

// Stop stops the certificate watcher
func (c *Cache) Stop() {
	close(c.stopCh)
	c.watcher.Close()

	log.Info("Certificate watcher stopped")
}
