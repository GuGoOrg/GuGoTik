package consul

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/utils/logging"
	"fmt"
	capi "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"
)

var consulClient *capi.Client

func init() {
	cfg := capi.DefaultConfig()
	cfg.Address = config.EnvCfg.ConsulAddr
	if c, err := capi.NewClient(cfg); err == nil {
		consulClient = c
		return
	} else {
		logging.Logger.Panicf("Connect Consul happens error: %v", err)
	}
}

func RegisterConsul(name string, port string) error {
	parsedPort, err := strconv.Atoi(port[1:]) // port start with ':' which like ':37001'
	logging.Logger.WithFields(log.Fields{
		"name": name,
		"port": parsedPort,
	}).Infof("Services Register Consul")

	if err != nil {
		return err
	}
	reg := &capi.AgentServiceRegistration{
		ID:   fmt.Sprintf("%s-1", name),
		Name: name,
		Port: parsedPort,
		Check: &capi.AgentServiceCheck{
			Interval:                       "5s",
			Timeout:                        "5s",
			GRPC:                           fmt.Sprintf("%s:%d/Heath", "127.0.0.1", parsedPort),
			DeregisterCriticalServiceAfter: "30s",
		},
	}
	if err := consulClient.Agent().ServiceRegister(reg); err != nil {
		return err
	}
	return nil
}

func ResolveService(serviceName string) (*capi.CatalogService, error) {
	for {
		instances, err := getServiceInstances(serviceName)
		if err != nil || len(instances) == 0 {
			logging.Logger.Panicf("Cannot find service: %s", serviceName)
		}

		selectedInstance := roundRobin(instances)
		if selectedInstance != nil {
			return selectedInstance, nil
		}
		time.Sleep(time.Second)
	}
}

func getServiceInstances(serviceName string) ([]*capi.CatalogService, error) {
	services, _, err := consulClient.Catalog().Service(serviceName, "", nil)
	if err != nil {
		return nil, err
	}

	return services, nil
}

func roundRobin(instances []*capi.CatalogService) *capi.CatalogService {
	if len(instances) == 0 {
		return nil
	}

	rand.NewSource(time.Now().UnixNano())
	index := rand.Intn(len(instances))

	return instances[index]
}
