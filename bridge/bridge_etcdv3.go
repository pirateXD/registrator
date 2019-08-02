package bridge

import (
	"github.com/pirateXD/registrator/vars"
	"log"
)

type XBridge struct {
	*Bridge
}

func (b *XBridge) Refresh() error {
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
		log.Println("refresh failed err:", err)
	} else {
		log.Println("refreshed:all")
	}

	return err
}

func (b *XBridge) SyncDockerList(err error) {
	//如果之前发生过错误(Register、Deregister、Refresh)， 并且租约正常，刷新Docker list
	if vars.GetLastErrCode() != nil && err == nil {
		log.Println("refresh trigger sync.")
		vars.SetLastErrCode(nil)
		b.Sync(true)
	}
}
