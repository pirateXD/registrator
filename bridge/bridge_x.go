package bridge

import (
	"github.com/pirateXD/registrator/etcdv3"
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

	_, isEtcV3 := b.registry.(*etcdv3.EtcdAdapter)
	if isEtcV3 {
		//etcd v3 需要特殊处理
		err := b.registry.Refresh(nil)
		if err != nil {
			log.Println("refresh failed:", err)
		}
		log.Println("refreshed:all")
	} else {
		//兼容其他的
		for containerId, services := range b.services {
			for _, service := range services {
				err := b.registry.Refresh(service)
				if err != nil {
					log.Println("refresh failed:", service.ID, err)
					continue
				}
				log.Println("refreshed:", containerId[:12], service.ID)
			}
		}
	}
}
