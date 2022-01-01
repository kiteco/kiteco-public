package api

import (
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/driver"
	"github.com/kiteco/kiteco/kite-golib/collections"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

type driversVal struct {
	ts     time.Time
	driver driver.Driver
}

type umfDrivers struct {
	cap int
	ttl time.Duration

	lock    *sync.Mutex
	drivers collections.OrderedMap
}

// newUMFDrivers initializes a new driver.Driver cache, keyed on UMF
// A capacity or ttl of 0 will disable use of that value to purge the cache.
func newUMFDrivers(capacity int, ttl time.Duration) umfDrivers {
	return umfDrivers{
		cap: capacity,
		ttl: ttl,

		lock:    &sync.Mutex{},
		drivers: collections.NewOrderedMap(capacity),
	}
}

func (m umfDrivers) Get(k data.UMF) driver.Driver {
	m.lock.Lock()
	defer m.lock.Unlock()

	kIface := (interface{})(k)
	vIface, ok := m.drivers.Delete(kIface)
	if !ok {
		vIface = &driversVal{driver: driver.New()}
	}
	vIface.(*driversVal).ts = time.Now().Add(m.ttl)

	m.purgeLocked(false) // purge before adding the driver back
	m.drivers.Set(kIface, vIface)

	return vIface.(*driversVal).driver
}

func (m umfDrivers) Reset() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.purgeLocked(true)
}

// purgeLocked deletes drivers from the cache.
// If all is true, the cache is cleared. Otherwise, the cache is purged according to m.cap/m.ttl.
// In the latter case, it clears enough drivers so that m.drivers.Len() < m.cap (strictly).
func (m umfDrivers) purgeLocked(all bool) {
	now := time.Now()
	m.drivers.RangeInc(func(k, v interface{}) bool {
		if all || (m.cap > 0 && m.drivers.Len() >= m.cap) || (m.ttl > 0 && now.Sub(v.(*driversVal).ts) > m.ttl) {
			m.drivers.Delete(k)
			v.(*driversVal).driver.Cleanup()
			return true
		}
		return false
	})
}
