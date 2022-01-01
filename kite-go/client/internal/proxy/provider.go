package proxy

import (
	"log"
	"net/url"
	"sync"
	"time"

	proxied "github.com/kiteco/go-get-proxied/proxy"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-golib/collections"
)

type provider interface {
	String() string
	ProxyForURL(url *url.URL) (*url.URL, error)
}

// -
type envProvider struct {
	cacheSize int
	cacheTTL  time.Duration

	provider proxied.Provider
	lock     *sync.Mutex
	cache    collections.OrderedMap
}

type cacheVal struct {
	ts  time.Time
	url *url.URL
}

func newEnvProvider() envProvider {
	return envProvider{
		cacheSize: 100,
		cacheTTL:  15 * time.Minute,

		provider: proxied.NewProvider(""),
		lock:     &sync.Mutex{},
		cache:    collections.NewOrderedMap(0),
	}
}

func (p envProvider) ProxyForURL(u *url.URL) (*url.URL, error) {
	urlStr := u.String()

	p.lock.Lock()
	defer p.lock.Unlock()

	now := time.Now()

	// check for a sufficiently recent cache entry
	if res, ok := p.cache.Get(urlStr); ok && now.Sub(res.(cacheVal).ts) < p.cacheTTL {
		if debug {
			log.Printf("using cached proxy URL: %v", res)
		}
		return res.(cacheVal).url, nil
	}

	// delete if it already exists
	p.cache.Delete(urlStr)

	// query the environment
	proxyURL, err := p.slowPath(u.Scheme, urlStr)
	if err != nil {
		return nil, err
	}

	// make space for the new cache entry
	p.cache.RangeInc(func(k, _ interface{}) bool {
		if p.cache.Len() < p.cacheSize {
			return false
		}
		p.cache.Delete(k)
		return true
	})

	// add the proxy URL to the cache
	p.cache.Set(urlStr, cacheVal{now, proxyURL})

	return proxyURL, nil
}

func (p envProvider) slowPath(scheme, url string) (*url.URL, error) {
	proxy := p.provider.GetProxy(scheme, url)
	if proxy == nil {
		if debug {
			log.Println("no proxy from env")
		}
		return nil, nil
	}

	proxyURL := proxy.URL()
	if debug {
		log.Printf("proxy URL from env: %s, %s, %s", proxyURL.String(), proxyURL.Scheme, proxyURL.Host)
	}
	return proxyURL, nil
}
func (p envProvider) String() string {
	return settings.EnvironmentProxySentinel
}

// -
type directProvider struct{}

func (p directProvider) ProxyForURL(url *url.URL) (*url.URL, error) {
	return nil, nil
}

func (p directProvider) String() string {
	return settings.NoProxySentinel
}

// -
type manualProvider struct {
	proxyURL *url.URL
}

func (p manualProvider) ProxyForURL(url *url.URL) (*url.URL, error) {
	if url.Scheme == "https" || url.Scheme == "http" {
		return p.proxyURL, nil
	}
	return nil, nil
}

func (p manualProvider) String() string {
	return p.proxyURL.String()
}
