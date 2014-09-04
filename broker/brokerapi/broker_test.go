package brokerapi_test

import (
    "github.com/brahmaroutu/docker-broker/broker/dockerapi"
    "github.com/brahmaroutu/docker-broker/broker/brokerapi"
    "github.com/brahmaroutu/docker-broker/broker/testhelpers"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "fmt"
    "time"
    "bytes"
    "strconv"
    "encoding/json"
    "net/http/httptest"
    "net/http"
    "net/url"
    "net"
    "io/ioutil"
)

var _ = Describe("Broker", func() {
    var brokerservice *dockerapi.DockerClient
    var serviceagent brokerapi.ServiceAgent
    var ts *httptest.Server
    var handler *testnet.TestHandler   
    var persister brokerapi.Persister
    var am *dockerapi.AgentManager
    var dispatcher brokerapi.DispatcherInterface
    var dockerServices []brokerapi.Service
    var opts brokerapi.Options
    var err error
    JustBeforeEach(func() {        
        opts = brokerapi.Options {
            Host: "localhost",
            Port: 61235,
            Username: "myFakeUser",
            Password: "myFakePassword",
            Debug: false,
            LogFile: "",
            Trace: false,
            PidFile: "",
        }
        
        config := testnet.BrokerConfiguration()
        
        serviceagent = testnet.NewServiceAgent() 
        persister = config.Persister
        persister.Connect()

        dockerServices = testnet.NewDockerServices()
        
        dispatcher = testnet.SimpleDispatcher()
        am,err = dockerapi.NewAgentManager(config,dispatcher)
        Expect(err).ShouldNot(HaveOccurred())
    
        broker := brokerapi.New(opts,am)
        go broker.Start()
        time.Sleep(4 * 1e9)
    })
    
    Describe("brokertest test broker interface", func() {
        BeforeEach(func() {        
            testnet.SetupSQL(persister)
        })
        AfterEach(func() {
            testnet.CleanupSQL(persister)
        })

        It("should publish catalog", func() {
            resp,respCode,err := SendHTTP("GET",BaseURL(opts)+"/v2/catalog",nil)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(200))
            var catalog brokerapi.Catalog
            json.Unmarshal(resp, &catalog)
            Expect(catalog.Services).Should(HaveLen(len(dockerServices)))
            Expect(catalog.Services).Should(ContainElement(dockerServices[0]))                    
            Expect(catalog.Services).Should(ContainElement(dockerServices[1]))                    
        })

        It("should fail to provision a service when no agents are available", func() {            
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_ListAllImagesRequest,testnet.Provision_CreateContainerRequest,testnet.Provision_InspectImageRequest,testnet.Provision_StartContainerRequest,testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            _,b := newProvisioningRequest()

            resp,respCode,err := SendHTTP("PUT",BaseURL(opts)+"/v2/service_instances/myFakeInstance",b)
            Expect(err).To(BeNil())

            var respmap map[string] interface{}
            json.Unmarshal(resp, &respmap)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusInternalServerError))
            Expect(respmap["description"]).To(Equal("no agents available"))
        })

        It("should provision a service", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_ListAllImagesRequest,testnet.Provision_CreateContainerRequest,testnet.Provision_InspectImageRequest,testnet.Provision_StartContainerRequest,testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            _,b := newProvisioningRequest()

            persister.AddServiceAgents([]brokerapi.ServiceAgent{serviceagent})    

            resp,respCode,err := SendHTTP("PUT",BaseURL(opts)+"/v2/service_instances/myFakeInstance",b)
            Expect(err).To(BeNil())

            var respmap map[string] interface{}
            json.Unmarshal(resp, &respmap)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusCreated))
            Expect(respmap["dashboard_url"]).To(Equal("mysql://fakehost:1234"))
        })

        It("should fail to deprovision when the specific agent is not present", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Deprovision_StopContainerRequest, testnet.Deprovision_RemoveContainerRequest})    
            defer ts.Close()

            pr,b := newProvisioningRequest()
            
            persister.AddServiceInstance("mysql:latest", 1234, 49153, 
                "mysql://fakehost:1234",  "myFakeInstance", "fakehost", "fakecontainename", "fakeimagename",
                pr, time.Now())    
            
            resp,respCode,err := SendHTTP("DELETE",BaseURL(opts)+"/v2/service_instances/myFakeInstance",b)
            Expect(err).To(BeNil())

            var respmap map[string] interface{}
            json.Unmarshal(resp, &respmap)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusGone))
            Expect(respmap["description"]).To(ContainSubstring("can't find agent - assume its already gone"))
        })

        It("should deprovision a service", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Deprovision_StopContainerRequest, testnet.Deprovision_RemoveContainerRequest})    
            defer ts.Close()

            pr,b := newProvisioningRequest()
            
            persister.AddServiceInstance("mysql:latest", 1234, 49153, 
                "mysql://fakehost:1234",  "myFakeInstance", "fakehost", "fakecontainename", "fakeimagename",
                pr, time.Now())    
                    
            persister.AddServiceAgents([]brokerapi.ServiceAgent{serviceagent})    

            resp,respCode,err := SendHTTP("DELETE",BaseURL(opts)+"/v2/service_instances/myFakeInstance",b)
            Expect(err).To(BeNil())

            var respmap map[string] interface{}
            json.Unmarshal(resp, &respmap)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusGone))
            Expect(respmap["description"]).To(ContainSubstring("can't find agent - assume its already gone"))
        })

        It("should fail to bind a service if agent is not available", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            pr,_ := newProvisioningRequest()
            _,b := newBindingRequest()
            
            persister.AddServiceInstance("mysql:latest", 1234, 49153, 
                "mysql://fakehost:1234",  "myFakeInstance", "fakehost", "fakecontainename", "fakeimagename",
                pr, time.Now())    
                    
            resp,respCode,err := SendHTTP("PUT",BaseURL(opts)+"/v2/service_instances/myFakeInstance/service_bindings/fakeBindingId",b)
            Expect(err).To(BeNil())

            var respmap map[string] interface{}
            json.Unmarshal(resp, &respmap)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusGone))
            Expect(respmap["description"]).To(ContainSubstring("can't find agent - assume its already gone"))
        })

        It("should fail to bind a service if service instance is missing", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            _,b := newBindingRequest()
        
            persister.AddServiceAgents([]brokerapi.ServiceAgent{serviceagent})    

            resp,respCode,err := SendHTTP("PUT",BaseURL(opts)+"/v2/service_instances/myFakeInstance/service_bindings/fakeBindingId",b)
            Expect(err).To(BeNil())

            var respmap map[string] interface{}
            json.Unmarshal(resp, &respmap)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusGone))
            Expect(respmap["description"]).To(ContainSubstring("Failed to find the service instance"))
        })


        It("should bind a service", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            pr,_ := newProvisioningRequest()
            _,b := newBindingRequest()
        
            persister.AddServiceAgents([]brokerapi.ServiceAgent{serviceagent})    

            persister.AddServiceInstance("mysql", 1234, 49153, 
                "mysql://fakehost:1234",  "myFakeInstance", serviceagent.DockerHost, "fakecontainename", "mysql",
                pr, time.Now())    
    
            resp,respCode,err := SendHTTP("PUT",BaseURL(opts)+"/v2/service_instances/myFakeInstance/service_bindings/fakeBindingId",b)
            Expect(err).To(BeNil())

            var respmap map[string] interface{}
            json.Unmarshal(resp, &respmap)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusCreated))
            
            fmt.Println(respmap)
            //Srini tempporary fix, may have to look into the tester app on how creds are parsed
            creds := respmap["credentials"].(map[string] interface{})

            Expect(creds["uri"]).To(Equal("mysql://fakehost:1234"))
            Expect(creds["host"]).To(Equal("fakeHost"))
            Expect(creds["port"]).To(Equal("1234"))
            Expect(creds["username"]).To(Equal("fakeUser"))
            Expect(creds["password"]).To(Equal("fakePassword"))
            Expect(creds["database"]).To(Equal("fakeDB"))
        })

        It("should fail to unbind a service if agent is not available", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            pr,_ := newProvisioningRequest()
            _,b := newBindingRequest()
            
            persister.AddServiceInstance("mysql:latest", 1234, 49153, 
                "mysql://fakehost:1234",  "myFakeInstance", "fakehost", "fakecontainename", "fakeimagename",
                pr, time.Now())    
                    
    
            resp,respCode,err := SendHTTP("DELETE",BaseURL(opts)+"/v2/service_instances/myFakeInstance/service_bindings/fakeBindingId",b)
            Expect(err).To(BeNil())

            var respmap map[string] interface{}
            json.Unmarshal(resp, &respmap)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusGone))
            Expect(respmap["description"]).To(ContainSubstring("can't find agent - assume its already gone"))
        })

        It("should fail to unbind a service if service instance is missing", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            _,b := newBindingRequest()
        
            persister.AddServiceAgents([]brokerapi.ServiceAgent{serviceagent})    

            resp,respCode,err := SendHTTP("DELETE",BaseURL(opts)+"/v2/service_instances/myFakeInstance/service_bindings/fakeBindingId",b)
            Expect(err).To(BeNil())

            var respmap map[string] interface{}
            json.Unmarshal(resp, &respmap)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusGone))
            Expect(respmap["description"]).To(ContainSubstring("Failed to find the service instance"))
        })

        It("should unbind a service", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerServiceWithMultipleRequests(serviceagent,persister, []testnet.TestRequest{testnet.Provision_InspectContainerRequest})    
            defer ts.Close()

            pr,_ := newProvisioningRequest()
            br,b := newBindingRequest()
        
            persister.AddServiceAgents([]brokerapi.ServiceAgent{serviceagent})    

            persister.AddServiceInstance("mysql", 1234, 49153, 
                "mysql://fakehost:1234",  "myFakeInstance", serviceagent.DockerHost,"fakecontainename", "mysql", 
                pr, time.Now())    

            persister.AddServiceBinding(br.InstanceId, br.BindingId, br.AppId, time.Now())

            _,respCode,err := SendHTTP("DELETE",BaseURL(opts)+"/v2/service_instances/myFakeInstance/service_bindings/fakeBindId",b)

            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(http.StatusOK))
            
        })


        It("should handle ping from service agent", func() {
            sa := testnet.NewServiceAgent()
            var b []byte
            b,err = json.Marshal(sa)
            Expect(err).To(BeNil())
            
            _,respCode,err := SendHTTP("POST",BaseURL(opts)+"/ping",b)
            Expect(err).To(BeNil())
            Expect(respCode).Should(Equal(200))
        })


    })

})

func BaseURL(opts brokerapi.Options) string {
    return "http://"+opts.Username+":"+opts.Password+"@"+opts.Host+":"+strconv.Itoa(opts.Port)
}

func SendHTTP(method,urlstr string, body []byte) ([]byte, int, error) {
    u, err := url.Parse(urlstr)
    if err != nil {
        return nil, -1, err
    }
    client := newHTTPClient(u)
    b := bytes.NewBuffer(body)
    req, err := http.NewRequest(method, urlstr, b)
    if err != nil {
        fmt.Println(err)
        return nil, -1, err
    }
    req.Header.Add("Content-Type", "application/json")
    req.Header.Add("X-Broker-Api-Version","2.2")
    resp, err := client.Do(req)
    if err != nil {
        return nil, http.StatusInternalServerError, err
    }
    defer resp.Body.Close()
    data, err := ioutil.ReadAll(resp.Body)
    if err != nil  {
        return nil, resp.StatusCode, err
    }
    return data, resp.StatusCode, err
}

func newHTTPClient(u *url.URL) *http.Client {
    httpTransport := &http.Transport{}
    if u.Scheme == "unix" {
        socketPath := u.Path
        unixDial := func(proto string, addr string) (net.Conn, error) {
            return net.Dial("unix", socketPath)
        }
        httpTransport.Dial = unixDial
        // Override the main URL object so the HTTP lib won't complain
        u.Scheme = "http"
        u.Host = "unix.sock"
    }
    u.Path = ""
    return &http.Client{Transport: httpTransport}
}

func newProvisioningRequest() (brokerapi.ProvisioningRequest, []byte) {
        pr := brokerapi.ProvisioningRequest {InstanceId: "myFakeInstance",
                ServiceId:  "mysql:latest",
                PlanId:     "100",
                OrgId:      "myFakeOrg",
                SpaceId:    "myFakeSpace",
            }            
        
        b,err := json.Marshal(pr)
        Expect(err).To(BeNil())
        
        return pr,b
}

func newBindingRequest() (brokerapi.BindingRequest, []byte) {
        br := brokerapi.BindingRequest {InstanceId: "myFakeInstance",
                BindingId:  "fakeBindId",
                ServiceId:  "mysql:latest",
                PlanId:     "100",
                AppId:      "myFakeApp",
            }            
        b,err := json.Marshal(br)
        Expect(err).To(BeNil())
        
        return br,b
}
            
