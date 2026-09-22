package config

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Interface wrapping viper.Viper type.
type IViper interface {
	AllSettings() map[string]interface{}
	Get(key string) interface{}
	GetBool(key string) bool
	GetDuration(key string) time.Duration
	GetFloat64(key string) float64
	GetInt(key string) int
	GetIntSlice(key string) []int
	GetUint64(key string) uint64
	GetUint64Slice(key string) []uint64
	GetString(key string) string
	GetStringSlice(key string) []string
	IsSet(key string) bool
	Set(key string, value interface{})
	SetDefault(key string, value interface{})
}

type SyncViper struct {
	mutex sync.RWMutex
	viper *viper.Viper
}

func NewSyncViper(viper *viper.Viper) IViper {
	return &SyncViper{
		viper: viper,
	}
}

func (syncViper *SyncViper) AllSettings() map[string]interface{} {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.AllSettings()
}

func (syncViper *SyncViper) Get(key string) interface{} {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.Get(key)
}

func (syncViper *SyncViper) GetBool(key string) bool {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.GetBool(key)
}

func (syncViper *SyncViper) GetDuration(key string) time.Duration {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.GetDuration(key)
}

func (syncViper *SyncViper) GetFloat64(key string) float64 {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.GetFloat64(key)
}

func (syncViper *SyncViper) GetInt(key string) int {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.GetInt(key)
}

func (syncViper *SyncViper) GetIntSlice(key string) []int {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.GetIntSlice(key)
}

func (syncViper *SyncViper) GetUint64(key string) uint64 {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.GetUint64(key)
}

func (syncViper *SyncViper) GetUint64Slice(key string) []uint64 {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()

	ss := syncViper.viper.GetStringSlice(key)
	out := make([]uint64, 0, len(ss))
	for _, s := range ss {
		v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (syncViper *SyncViper) GetString(key string) string {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.GetString(key)
}

func (syncViper *SyncViper) GetStringSlice(key string) []string {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.GetStringSlice(key)
}

func (syncViper *SyncViper) IsSet(key string) bool {
	syncViper.mutex.RLock()
	defer syncViper.mutex.RUnlock()
	return syncViper.viper.IsSet(key)
}

func (syncViper *SyncViper) Set(key string, value interface{}) {
	syncViper.mutex.Lock()
	defer syncViper.mutex.Unlock()
	syncViper.viper.Set(key, value)
}

func (syncViper *SyncViper) SetDefault(key string, value interface{}) {
	syncViper.mutex.Lock()
	defer syncViper.mutex.Unlock()
	syncViper.viper.SetDefault(key, value)
}
