package dockerapi

import (
    "errors"
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/brokerapi"
    "math/rand"
)

type SimpleDispatcher struct {
    config BrokerConfiguration
}

func NewSimpleDispatcher(config BrokerConfiguration) *SimpleDispatcher {
    config.Persister.Connect()    
    return &SimpleDispatcher{config}
}

func (sd *SimpleDispatcher) NewBrokerService() (brokerapi.BrokerService, error) {
    serviceagents, err := sd.config.Persister.GetServiceAgentList(
        sd.config.Persister.TimeElapsed("last_ping") + " < 3*ping_interval_secs and perf_factor=(select min(perf_factor) from serviceagents)")
    if err != nil {
        return nil, err
    }
    if len(serviceagents) == 0 {
        return nil, errors.New("no agents available")
    }
    var index = rand.Intn(len(serviceagents))
    dockerclient, err := NewDockerClient(serviceagents[index],sd.config)
    return dockerclient, err
}
