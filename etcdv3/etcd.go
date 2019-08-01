package etcdv3

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/pirateXD/registrator/bridge"
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

	return &EtcdAdapter{client: newClient(urls), path: uri.Path}
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
		r.leaseID = resp.ID
		return nil
	}
}

func (r *EtcdAdapter) KeepAliveOnce(ttl int) error {
	if r.leaseID <= 0 {
		log.Println("EtcdAdapter KeepAliveOnce:", "etcd leaseID:", r.leaseID)
		if err := r.GrantLease(ttl); err != nil {
			return err
		}
	} else {
		_, err := r.client.KeepAliveOnce(context.TODO(), r.leaseID)
		if err == rpctypes.ErrLeaseNotFound {
			log.Println("EtcdAdapter KeepAliveOnce: ErrLeaseNotFound")
			if err := r.GrantLease(ttl); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (r *EtcdAdapter) Register(service *bridge.Service) error {
	r.syncEtcdCluster()

	path := r.etcKey(service)
	port := strconv.Itoa(service.Port)
	addr := net.JoinHostPort(service.IP, port)

	err := r.KeepAliveOnce(service.TTL)
	if err != nil {
		log.Println("etcd: failed to register service KeepAliveOnce:", err)
		return err
	}

	if _, err := r.client.Put(context.TODO(), path, addr, clientv3.WithLease(r.leaseID)); err != nil {
		log.Println("etcd: failed to register service put:", err)
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
	}
	return err
}

func (r *EtcdAdapter) Refresh(service *bridge.Service) error {
	r.syncEtcdCluster()
	err := r.KeepAliveOnce(service.TTL)
	if err != nil {
		log.Println("etcd: failed to refresh service KeepAliveOnce:", err)
		return err
	}

	return nil
}

func (r *EtcdAdapter) Services() ([]*bridge.Service, error) {
	return []*bridge.Service{}, nil
}
