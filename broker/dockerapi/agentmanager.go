package dockerapi

import (
    "github.com/brahmaroutu/docker-broker/broker/brokerapi"
    "log"
)

type AgentManager struct {
    config     BrokerConfiguration
    Dispatcher brokerapi.DispatcherInterface
}

func NewAgentManager(config BrokerConfiguration, dispatcher brokerapi.DispatcherInterface) (*AgentManager, error) {
    config.Persister.Connect()
    return &AgentManager{config, dispatcher}, nil
}

func (am *AgentManager) Ping(sa brokerapi.ServiceAgent) error {
    err := am.config.Persister.AddorUpdateServiceAgent(sa)
    if err != nil {
        return err
    }
    return nil
}

func (am *AgentManager) GetServiceAgent(instanceid string) (brokerapi.BrokerService, error) {
    var brokerservice brokerapi.BrokerService
    var err error
    if len(instanceid) == 0 {
        //use DISPATCHER to pick the ServiceAgent from persister
        brokerservice, err = am.Dispatcher.NewBrokerService()
        if err != nil {
            return nil, err
        }
    } else {
        //look for service agent that holds this instance
        serviceagent, err := am.config.Persister.GetServiceAgentFromInstance("cf_instance_id='" + instanceid + "'")
        log.Println("GetServiceAgentFromInstance cf_instance_id='" + instanceid + "'",serviceagent)
        if err != nil {
            return nil, brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeGone,"Failed to find the service instance ("+err.Error()+")"})
        }
        serviceagents, err := am.config.Persister.GetServiceAgentList("docker_host='" + serviceagent + "'")
        log.Println("GetServiceAgentList host='" + serviceagent + "'",serviceagents)
        if len(serviceagents) == 0 {
            return nil, brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeGone, "Handler: can't find agent - assume its already gone"})
        }
        if err != nil {
            return nil, brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeGone, err.Error()})
        }

        brokerservice, err = NewDockerClient(serviceagents[0],am.config)
    }

    //handler should recycle this dockerclient, may be we should deref at the end of handler code ot simply call obj.close()?
    return brokerservice, err
}

func (am *AgentManager) Catalog() (brokerapi.Catalog, error) {
    return brokerapi.Catalog{am.config.GetServices()}, nil
}

func (am *AgentManager) AddImage(catalog string, img brokerapi.ImageDefinition) error {
    return am.config.AddOrUpdateImageDefinition(catalog,img)
}

func (am *AgentManager) GetImage(catalog,name string) ([]brokerapi.ImageDefinition,error) {
    return am.config.GetImageDefinitions(name)
}

func (am *AgentManager) DeleteImage(catalog,name string) error {
    return am.config.DeleteImageDefinition(catalog,name)
}

func (am *AgentManager) AddCerts(certs brokerapi.BrokerCerts) error {
    return am.config.AddOrUpdateCertificates(certs)
}

func (am *AgentManager) GetCerts(host string) ([]brokerapi.BrokerCerts, error) {
    return am.config.GetCertificates(host)
}

func (am *AgentManager) DeleteCerts(host string) error {
    return am.config.DeleteCertificate(host)
}
