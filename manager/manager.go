package manager

import (
	"sync"

	"github.com/abema/antares/core"
)

type Config struct {
	AutoRemove bool
}

type Manager interface {
	Add(id string, config *core.Config) bool
	Remove(id string) bool
	RemoveAll() []string
	Batch(map[string]*core.Config) (added, removed []string)
	Get(id string) core.Monitor
	Map() map[string]core.Monitor
}

func NewManager(config *Config) Manager {
	return &manager{
		config:   config,
		monitors: make(map[string]core.Monitor),
	}
}

type manager struct {
	config   *Config
	monitors map[string]core.Monitor
	mutex    sync.RWMutex
}

func (m *manager) Add(id string, config *core.Config) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.add(id, config)
}

func (m *manager) add(id string, config *core.Config) bool {
	if _, exists := m.monitors[id]; exists {
		return false
	}
	if m.config != nil && m.config.AutoRemove {
		orgOnTerminate := config.OnTerminate
		copied := *config
		config = &copied
		config.OnTerminate = func() {
			m.Remove(id)
			if orgOnTerminate != nil {
				orgOnTerminate()
			}
		}
	}
	m.monitors[id] = core.NewMonitor(config)
	return true
}

func (m *manager) Remove(id string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.remove(id)
}

func (m *manager) remove(id string) bool {
	monitor, exists := m.monitors[id]
	if !exists {
		return false
	}
	delete(m.monitors, id)
	go func() {
		monitor.Terminate()
	}()
	return true
}

func (m *manager) RemoveAll() []string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	removed := make([]string, 0, 4)
	for id := range m.monitors {
		m.remove(id)
		removed = append(removed, id)
	}
	return removed
}

func (m *manager) Batch(configs map[string]*core.Config) (added, removed []string) {
	added = make([]string, 0, 4)
	removed = make([]string, 0, 4)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for id := range m.monitors {
		if _, ok := configs[id]; !ok {
			m.remove(id)
			removed = append(removed, id)
		}
	}
	for id, config := range configs {
		if m.add(id, config) {
			added = append(added, id)
		}
	}
	return
}

func (m *manager) Get(id string) core.Monitor {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.monitors[id]
}

func (m *manager) Map() map[string]core.Monitor {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	copied := make(map[string]core.Monitor, len(m.monitors))
	for id, monitor := range m.monitors {
		copied[id] = monitor
	}
	return copied
}
