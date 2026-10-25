package aws

import (
	"strconv"
	"strings"
	"sync"
)

// clientFactory builds a Client from credentials. Defaults to NewClient;
// overridable in tests to count and control client construction.
type clientFactory func(Credentials) (Client, error)

type clientEntry struct {
	once   sync.Once
	client Client
	err    error
}

// clientCache builds at most one Client per distinct Credentials value and
// shares it across all runners. Safe for concurrent use.
type clientCache struct {
	factory clientFactory
	mu      sync.Mutex
	entries map[credentialsKey]*clientEntry
}

func newClientCache(factory clientFactory) *clientCache {
	return &clientCache{factory: factory, entries: map[credentialsKey]*clientEntry{}}
}

func (c *clientCache) get(creds Credentials) (Client, error) {
	key := creds.cacheKey()
	c.mu.Lock()
	entry, ok := c.entries[key]
	if !ok {
		entry = &clientEntry{}
		c.entries[key] = entry
	}
	c.mu.Unlock()

	entry.once.Do(func() {
		entry.client, entry.err = c.factory(creds)
	})

	if entry.err != nil {
		// Do not cache failures: drop the entry so a later call retries.
		c.mu.Lock()
		if c.entries[key] == entry {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil, entry.err
	}
	return entry.client, nil
}

type credentialsKey struct {
	accessKey   string
	secretKey   string
	profile     string
	credsFile   string
	assumeRoles string
	region      string
}

func (c Credentials) cacheKey() credentialsKey {
	var assumeRoles strings.Builder
	for _, role := range c.AssumeRoles {
		writeCacheKeyPart(&assumeRoles, role.RoleARN)
		writeCacheKeyPart(&assumeRoles, role.ExternalID)
		writeCacheKeyPart(&assumeRoles, role.Policy)
		writeCacheKeyPart(&assumeRoles, role.SessionName)
	}

	return credentialsKey{
		accessKey:   c.AccessKey,
		secretKey:   c.SecretKey,
		profile:     c.Profile,
		credsFile:   c.CredsFile,
		assumeRoles: assumeRoles.String(),
		region:      c.Region,
	}
}

func writeCacheKeyPart(key *strings.Builder, value string) {
	key.WriteString(strconv.Itoa(len(value)))
	key.WriteByte(':')
	key.WriteString(value)
}
