package etcdv3

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/pirateXD/registrator/bridge"
	"github.com/pirateXD/registrator/vars"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func init() {
	bridge.Register(new(Factory), "etcd")
}

type Factory struct{}

func newClient(host []string) *clientv3.Client {
	// config
	cfg := clientv3.Config{
		Endpoints:            host,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    time.Second,
		DialKeepAliveTimeout: time.Second,
	}

	// create client
	cli, err := clientv3.New(cfg)
	if err != nil {
		panic(err)
	}
	return cli
}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
	urls := make([]string, 0)
	if uri.Host != "" {
		urls = append(urls, "http://"+uri.Host)
	} else {
		urls = append(urls, "http://127.0.0.1:2379")
	}

	res, err := http.Get(urls[0] + "/version")
	if err != nil {
		log.Fatal("etcd: error retrieving version", err)
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	log.Printf("etcd version response : %v", body)

	r := &EtcdAdapter{client: newClient(urls), path: uri.Path}
	err = r.GrantLease(vars.ConfigTTL)
	if err != nil {
		log.Fatal("etcd:  New EtcdAdapter  GrantLease error:", err)
	}
	return r
}

type EtcdAdapter struct {
	client  *clientv3.Client
	path    string
	leaseID clientv3.LeaseID //共用一个租期即可
}

func (r *EtcdAdapter) Ping() error {
	r.syncEtcdCluster()

	var err error
	_, err = r.client.MemberList(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func (r *EtcdAdapter) syncEtcdCluster() {
	var result = r.client.Sync(context.TODO())
	if nil != result {
		log.Println("etcd: sync cluster was unsuccessful")
	}
}

func (r *EtcdAdapter) GrantLease(ttl int) error {
	if resp, err := r.client.Grant(context.TODO(), int64(ttl)); err != nil {
		log.Println("etcd: failed to GrantLease:", err)
		return err
	} else {
		log.Println("GrantLease success old leaseId:", r.leaseID, " new leaseId:", resp.ID)
		r.leaseID = resp.ID
		return nil
	}
}

func (r *EtcdAdapter) KeepAliveOnce() error {
	if r.leaseID <= 0 {
		log.Println("EtcdAdapter KeepAliveOnce etcd leaseID:", r.leaseID)
		return rpctypes.ErrLeaseNotFound
	} else {
		//获得etcd租约剩余时间
		//timeliveResp, _ := r.client.TimeToLive(context.TODO(), r.leaseID)
		//log.Println("EtcdAdapter KeepAliveOnce timeliveResp.TTL:", timeliveResp.TTL, "timeliveResp.GrantedTTL", timeliveResp.GrantedTTL)

		_, err := r.client.KeepAliveOnce(context.TODO(), r.leaseID)
		return err
	}
	return nil
}

func (r *EtcdAdapter) Register(service *bridge.Service) error {
	r.syncEtcdCluster()

	path := r.etcKey(service)
	port := strconv.Itoa(service.Port)
	addr := net.JoinHostPort(service.IP, port)

	//租约过期不处理，下次refresh 会刷新docker list.
	//如果创建新租约，新的servcie会registrer成功，但是已经过期的service不会更新，会信息丢失.
	err := r.KeepAliveOnce()
	if err != nil {
		log.Println("etcd: failed to register service KeepAliveOnce:", err)
		vars.SetLastErrCode(err)
		return err
	}

	if _, err := r.client.Put(context.TODO(), path, addr, clientv3.WithLease(r.leaseID)); err != nil {
		log.Println("etcd: failed to register service put:", err)
		vars.SetLastErrCode(err)
		return err
	}
	return nil
}

func (r *EtcdAdapter) etcKey(service *bridge.Service) string {
	path := r.path + "/" + service.Name + "/" + service.ID
	return path
}

func (r *EtcdAdapter) Deregister(service *bridge.Service) error {
	r.syncEtcdCluster()

	path := r.etcKey(service)
	var err error
	_, err = r.client.Delete(context.TODO(), path)
	if err != nil {
		log.Println("etcd: failed to deregister service:", err)
		vars.SetLastErrCode(err)
	}
	return err
}

func (r *EtcdAdapter) Refresh(service *bridge.Service) error {
	r.syncEtcdCluster()
	err := r.KeepAliveOnce()
	if err != nil {
		vars.SetLastErrCode(err)
	}

	if err == rpctypes.ErrLeaseNotFound {
		//如果租约失效, 重新获得租约
		log.Println("EtcdAdapter KeepAliveOnce: ErrLeaseNotFound  do GrantLease")
		err = r.GrantLease(vars.ConfigTTL)
	}

	if err != nil {
		log.Println("etcd: failed to refresh service KeepAliveOnce:", err)
		return err
	}

	return nil
}

func (r *EtcdAdapter) Services() ([]*bridge.Service, error) {
	return []*bridge.Service{}, nil
}
