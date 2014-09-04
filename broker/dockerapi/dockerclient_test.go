package dockerapi_test

import (
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/dockerapi"
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/brokerapi"
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/testhelpers"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "fmt"
    "time"
    "net/http"
    "net/http/httptest"
)

var _ = Describe("Dockerclient", func() {
    var brokerservice *dockerapi.DockerClient
    var serviceagent brokerapi.ServiceAgent
    var persister brokerapi.Persister
    var cm dockerapi.BrokerConfiguration
    var ts *httptest.Server
    var handler *testnet.TestHandler   
    var err error
    
    Describe("brokertest test dockerclient", func() {
        BeforeEach(func() {        
            serviceagent = testnet.NewServiceAgent() 
            persister = testnet.NewPersister()
            cm = testnet.BrokerConfiguration()
            testnet.SetupSQL(persister)
        })
        AfterEach(func() {
            testnet.CleanupSQL(persister)
        })
        
        It("create docker client", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerService(serviceagent,persister, emptyGetRequest)    
            defer ts.Close()
            
            Expect(brokerservice).ShouldNot(BeNil())
            Expect("http://"+brokerservice.URL.Host).To(Equal(ts.URL))
            Expect(brokerservice.HTTPClient).ShouldNot(BeNil())
        })

        It("should not respond to catalog request", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerService(serviceagent,persister, emptyGetRequest)    
            defer ts.Close()

            var catalog brokerapi.Catalog
            catalog, err = brokerservice.Catalog()
            Expect(catalog.Services).To(BeEmpty())
            Expect(err).Should(BeAssignableToTypeOf(&dockerapi.CFError{}))
            Expect(err.Error()).To(Equal("Catalog is not Supported from the client"))
        })

        It("should provision a service", func() {
            persister.Connect()
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_ListAllImagesRequest,testnet.Provision_CreateContainerRequest,testnet.Provision_InspectImageRequest,testnet.Provision_StartContainerRequest,testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            var pr brokerapi.ProvisioningRequest
            var provisionurl string
            pr = brokerapi.ProvisioningRequest {InstanceId: "myFakeInstance",
                    ServiceId:  "mysql:latest",
                    PlanId:     "100",
                    OrgId:      "myFakeOrg",
                    SpaceId:    "myFakeSpace",
                }            
            provisionurl, err = brokerservice.Provision(pr)
            Expect(err).To(BeNil())
            Expect(provisionurl).To(Equal("mysql://fakehost:1234"))
        })

        It("should bind a service", func() {
            persister.Connect()
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            var br brokerapi.BindingRequest
            br = brokerapi.BindingRequest {InstanceId: "myFakeInstance",
                    BindingId:  "fakeBindId",
                    ServiceId:  "mysql:latest",
                    PlanId:     "100",
                    AppId:      "myFakeApp",
                }            
            pr := brokerapi.ProvisioningRequest {InstanceId: "myFakeInstance",
                    ServiceId:  "mysql:latest",
                    PlanId:     "100",
                    OrgId:      "myFakeOrg",
                    SpaceId:    "myFakeSpace",
                }            
            
            persister.AddServiceInstance("mysql:latest", 1234, 49153, 
            "mysql://fakehost:1234",  "myFakeInstance", "fakehost", 
            "myFakeContainer", "mysql", pr, time.Now())    
            
            _,creds,somestr,err := brokerservice.Bind(br)
            Expect(err).To(BeNil())
            fmt.Println(creds)
            Expect(creds["uri"]).To(Equal("mysql://fakehost:1234"))
            Expect(creds["host"]).To(Equal("fakeHost"))
            Expect(creds["port"]).To(Equal("1234"))
            Expect(creds["username"]).To(Equal("fakeUser"))
            Expect(creds["password"]).To(Equal("fakePassword"))
            Expect(creds["database"]).To(Equal("fakeDB"))
            Expect(somestr).To(Equal("unknown"))
        })


        It("should unbind a service", func() {
            persister.Connect()
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Deprovision_StopContainerRequest, testnet.Deprovision_RemoveContainerRequest})    
            defer ts.Close()

            var br brokerapi.BindingRequest
            br = brokerapi.BindingRequest {InstanceId: "myFakeInstance",
                    BindingId:  "fakeBindId",
                    ServiceId:  "mysql:latest",
                    PlanId:     "100",
                    AppId:      "myFakeApp",
                }            
            pr := brokerapi.ProvisioningRequest {InstanceId: "myFakeInstance",
                    ServiceId:  "mysql:latest",
                    PlanId:     "100",
                    OrgId:      "myFakeOrg",
                    SpaceId:    "myFakeSpace",
                }            
            
            persister.AddServiceInstance("mysql:latest", 1234, 49153, 
            "mysql://fakehost:1234", "myFakeContainerId", "fakehost", 
            "myFakeContainer", "mysql", pr, time.Now())    
            persister.AddServiceBinding(br.InstanceId, br.BindingId, br.AppId, time.Now())
            
            err := brokerservice.Unbind(br)
            Expect(err).To(BeNil())
        })


        It("should deprovision a service", func() {
            persister.Connect()
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Deprovision_StopContainerRequest, testnet.Deprovision_RemoveContainerRequest})    
            defer ts.Close()

            pr := brokerapi.ProvisioningRequest {InstanceId: "myFakeInstance",
                    ServiceId:  "mysql:latest",
                    PlanId:     "100",
                    OrgId:      "myFakeOrg",
                    SpaceId:    "myFakeSpace",
                }            
            
            persister.AddServiceInstance("mysql:latest", 1234, 49153, 
            "mysql://fakehost:1234", "myFakeInstance", "fakehost", 
            "myFakeContainer", "mysql", pr, time.Now())    
            
            err := brokerservice.Deprovision(pr)
            Expect(err).To(BeNil())
        })

    })

})
    
    
var emptyGetRequest = testnet.NewTestRequest(testnet.TestRequest{
    Method:  "GET",
    Path:    "/",
    Response: testnet.TestResponse{
        Status: http.StatusOK,
    },
})    
