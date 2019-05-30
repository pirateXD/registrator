package bridge

import (
	"log"
)

type XBridge struct {
	*Bridge
}

func (b *XBridge) Refresh() {
	b.Lock()
	defer b.Unlock()

	for containerId, deadContainer := range b.deadContainers {
		deadContainer.TTL -= b.config.RefreshInterval
		if deadContainer.TTL <= 0 {
			delete(b.deadContainers, containerId)
		}
	}

	//etcd v3 需要特殊处理
	err := b.registry.Refresh(nil)
	if err != nil {
		log.Println("refresh failed:", err)
	}
	log.Println("refreshed:all")
}
