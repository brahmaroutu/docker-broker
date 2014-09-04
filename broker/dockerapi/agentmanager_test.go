package dockerapi_test

import (
    "github.com/brahmaroutu/docker-broker/broker/dockerapi"
    "github.com/brahmaroutu/docker-broker/broker/brokerapi"
    "github.com/brahmaroutu/docker-broker/broker/testhelpers"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "time"
    "errors"
)

var _ = Describe("Agentmanager", func() {
    var persister brokerapi.Persister
    var am *dockerapi.AgentManager
    var dockerServices []brokerapi.Service
    var cm dockerapi.BrokerConfiguration
    var err error
    JustBeforeEach(func() {
        persister = testnet.NewPersister()
        persister.Connect()
        cm = testnet.BrokerConfiguration()

        dispatcher := testnet.SimpleDispatcher()
        dockerServices = testnet.NewDockerServices()
        
        am,err = dockerapi.NewAgentManager(testnet.BrokerConfiguration(),dispatcher)
        Expect(err).ShouldNot(HaveOccurred())
    
    })
    
    Describe("brokertest test simpledispatcher", func() {
        BeforeEach(func() {        
            testnet.SetupSQL(persister)
        })
        AfterEach(func() {
            testnet.CleanupSQL(persister)
        })

        It("able to receive a ping", func() {
            sa := brokerapi.ServiceAgent {
                ServiceHost: "myFakeHost",
                DockerHost: "myFakeHost",
                DockerPort:  1234,
                LastPing: time.Now(),
                IsActive: true,
                PerfFactor: 1.0,
                KeepAlive: 1,
            }

            err = am.Ping(sa)
            Expect(err).To(BeNil())
        })

        It("able to receive a successive pings", func() {
            sa := testnet.NewServiceAgent()
            sa.ServiceHost = "myFakeHost"

            err = am.Ping(sa)
            Expect(err).To(BeNil())

            err = am.Ping(sa)
            Expect(err).To(BeNil())
        })
        
        It("failes to provide serviceagent when ping is not received", func() {
            _,err = am.GetServiceAgent("")
            Expect(err).To(Equal(errors.New("no agents available")))
        })

        It("provides a serviceagent when ping is received", func() {
            sa := testnet.NewServiceAgent()
            sa.ServiceHost = "myFakeHost"

            err = am.Ping(sa)
            var newbrokerservice brokerapi.BrokerService
            newbrokerservice,err = am.GetServiceAgent("")
            Expect(err).To(BeNil())
            Expect(newbrokerservice).ShouldNot(BeNil())
        })

        It("fail to return specific serviceagent", func() {
            sa := testnet.NewServiceAgent()
            sa.ServiceHost = "myFakeHost"

            err = am.Ping(sa)
            var newbrokerservice brokerapi.BrokerService
            newbrokerservice,err = am.GetServiceAgent("myFakeInstance")
            Expect(err).To(Equal(&dockerapi.CFError{ErrorCode:20, ErrorDesc:"Failed to find the service instance (sql: no rows in result set)"}))
            Expect(newbrokerservice).Should(BeNil())
        })
        
        
        It("should return a specific serviceagent", func() {

            sa := testnet.NewServiceAgent()
            sa.DockerHost = "myFakeHost"
            
            pr := brokerapi.ProvisioningRequest {InstanceId: "myFakeInstance",
                    ServiceId:  "mysql:latest",
                    PlanId:     "100",
                    OrgId:      "myFakeOrg",
                    SpaceId:    "myFakeSpace",
                }            
                
            persister.AddServiceInstance("mysql:latest", 1234, 49153, 
                "mysql://fakehost:1234", "myFakeInstance", "myFakeHost", "myFakeContainer", "mysql", 
                pr, time.Now())    
                        
            err = am.Ping(sa)
            var newbrokerservice brokerapi.BrokerService
            newbrokerservice,err = am.GetServiceAgent("myFakeInstance")
            Expect(err).To(BeNil())
            Expect(newbrokerservice).ShouldNot(BeNil())
        })
        
        
        It("should return a catalog list", func() {
            catalog,err := am.Catalog()
            Expect(err).To(BeNil())
            Expect(catalog.Services).Should(HaveLen(len(dockerServices)))
            Expect(catalog.Services).Should(ContainElement(dockerServices[0]))                    
            Expect(catalog.Services).Should(ContainElement(dockerServices[1]))                    
        })                
    })
})
