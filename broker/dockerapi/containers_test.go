package dockerapi_test

import (
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/dockerapi"
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/brokerapi"
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/testhelpers"
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    "net/http"
    "net/http/httptest"
)

var _ = Describe("Containers", func() {
    var Cconfig  dockerapi.ContainerConfig
    var brokerservice *dockerapi.DockerClient
    var serviceagent brokerapi.ServiceAgent
    var persister brokerapi.Persister
    var ts *httptest.Server
    var handler *testnet.TestHandler   
    var err error
    JustBeforeEach(func() {
           var vols,exposedports map[string] struct{}
           Cconfig = dockerapi.ContainerConfig {
            Hostname:"",
            User:"",
            Memory:0,
            MemorySwap:0,
            AttachStdin:false,
            AttachStdout:true,
            AttachStderr:true,
            PortSpecs:nil,
            Tty:false,
            OpenStdin:false,
            StdinOnce:false,
            Env:nil,
            Cmd: []string {
                  "date",
              },
              Image:"base",
              Volumes: vols,
              WorkingDir:"",
              ExposedPorts: exposedports,
        }    
        serviceagent = testnet.NewServiceAgent() 
        persister = testnet.NewPersister()

    })
    
    Describe("brokertest test containers", func() {
        It("should create container", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerService(serviceagent, persister, createContainerRequest)    
            defer ts.Close()    

            var containerid string
            containerid,err = Cconfig.CreateContainer(*brokerservice,"FakeName")
            Expect(err).To(BeNil())
            Expect(containerid).To(Equal("myFakeContainerId"))
        })

        It("should start container", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerService(serviceagent, persister, startContainerRequest)    
            defer ts.Close()    
            hostConfig := dockerapi.HostConfig{ContainerIDFile: "FakeFile", PublishAllPorts: true, NetworkMode: "bridge"}
            err = brokerservice.StartContainer("myFakeContainerId",&hostConfig)
            Expect(err).To(BeNil())
        })    
        
        It("should stop container", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerService(serviceagent, persister, stopContainerRequest)    
            defer ts.Close()    

            err = brokerservice.StopContainer("myFakeContainerId",22)
            Expect(err).To(BeNil())
        })    

        It("should remove container", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerService(serviceagent, persister, removeContainerRequest)    
            defer ts.Close()    

            err = brokerservice.RemoveContainer("myFakeContainerId")
            Expect(err).To(BeNil())
        })    

        It("should inspect container", func() {
            brokerservice,serviceagent,ts,handler = testnet.NewBrokerService(serviceagent, persister, inspectContainerRequest)    
            defer ts.Close()    

            ports := make( map[string] []dockerapi.PortBinding)
            ports["3306/tcp"] = []dockerapi.PortBinding { dockerapi.PortBinding{HostIp: "0.0.0.0", HostPort: "49153"}}

            containerinfo, err := brokerservice.InspectContainer("myFakeContainerId")
            Expect(err).To(BeNil())
            Expect(containerinfo.Id).To(Equal("myFakeContainerId"))
            Expect(containerinfo.Config.Hostname).To(Equal("myFakeHost"))
            Expect(containerinfo.State.Running).To(Equal(true))
            Expect(containerinfo.NetworkSettings.Ports).To(Equal(ports))
        })    

    })
})

var createContainerRequest = testnet.NewTestRequest(testnet.TestRequest{
    Method:  "POST",
    Path:    "/containers/create?name=FakeName",
    Matcher: testnet.RequestBodyMatcher(`{"User": "","Memory": 0,"PortSpecs": null,"StdinOnce": false,"Image": "base","Domainname": "","Cpuset": "","AttachStderr": true,"ExposedPorts": null,"Tty": false,"Cmd": ["date"],"MemorySwap": 0,"CpuShares": 0,"AttachStdin": false,"OpenStdin": false,"WorkingDir": "","NetworkDisabled": false,"OnBuild": null,"Hostname": "","AttachStdout": true,"Env": null,"Volumes": null,"Entrypoint": null}`),
    Response: testnet.TestResponse{
        Status: http.StatusOK,
        Body : `{
            "Id":"myFakeContainerId",
            "Warnings":[]
        }`,
    },

})    

var startContainerRequest = testnet.NewTestRequest(testnet.TestRequest{
    Method:  "POST",
    Path:    "/containers/myFakeContainerId/start",
    Matcher: testnet.RequestBodyMatcher(`{"ContainerIDFile": "FakeFile","Privileged": false,"PublishAllPorts": true,"NetworkMode": "bridge","Binds": null,"PortBindings": null,"Links": null,"Dns": null,"DnsSearch": null,"VolumesFrom": null,"LxcConf": null, "RestartPolicy": {"Name": "","MaximumRetryCount": 0}}`),
    Response: testnet.TestResponse{
        Status: http.StatusOK,
    },
})    

var stopContainerRequest = testnet.NewTestRequest(testnet.TestRequest{
    Method:  "POST",
    Path:    "/containers/myFakeContainerId/stop?t=22",
    Response: testnet.TestResponse{
        Status: http.StatusOK,
    },
})    

var removeContainerRequest = testnet.NewTestRequest(testnet.TestRequest{
    Method:  "DELETE",
    Path:    "/containers/myFakeContainerId",
    Response: testnet.TestResponse{
        Status: http.StatusOK,
    },
})    

var inspectContainerRequest = testnet.NewTestRequest(testnet.TestRequest{
    Method:  "GET",
    Path:    "/containers/myFakeContainerId/json",
    Response: testnet.TestResponse{
        Status: http.StatusOK,
        Body: `{
                 "Id": "myFakeContainerId",
                 "Created": "2013-05-07T14:51:42.041847+02:00",
                 "Path": "date",
                 "Args": [],
                 "Config": {
                         "Hostname": "myFakeHost",
                         "User": "",
                         "Memory": 0,
                         "MemorySwap": 0,
                         "AttachStdin": false,
                         "AttachStdout": true,
                         "AttachStderr": true,
                         "PortSpecs": null,
                         "Tty": false,
                         "OpenStdin": false,
                         "StdinOnce": false,
                         "Env": null,
                         "Cmd": [
                                 "date"
                         ],
                         "Dns": null,
                         "Image": "base",
                         "Volumes": {},
                         "VolumesFrom": "",
                         "WorkingDir":""

                 },
                 "State": {
                         "Running": true,
                         "Pid": 0,
                         "ExitCode": 0,
                         "StartedAt": "2013-05-07T14:51:42.087658+02:01360",
                         "Ghost": false
                 },
                 "Image": "b750fe79269d2ec9a3c593ef05b4332b1d1a02a62b4accb2c21d589ff2f5f2dc",
                 "NetworkSettings": {
                         "IpAddress": "",
                         "IpPrefixLen": 0,
                         "Gateway": "",
                         "Bridge": "",
                         "PortMapping": null,
                         "Ports": {
                             "3306/tcp": [
                             {
                                  "HostIp": "0.0.0.0",
                                  "HostPort": "49153"
                             }
                             ]
                         }
                 },
                 "SysInitPath": "/home/kitty/go/src/github.com/dotcloud/docker/bin/docker",
                 "ResolvConfPath": "/etc/resolv.conf",
                 "Volumes": {},
                 "HostConfig": {
                     "Binds": null,
                     "ContainerIDFile": "",
                     "LxcConf": [],
                     "Privileged": false,
                     "PortBindings": {
                        "80/tcp": [
                            {
                                "HostIp": "0.0.0.0",
                                "HostPort": "49153"
                            }
                        ]
                     },
                     "Links": ["/name:alias"],
                     "PublishAllPorts": false
                 }
        }`,
    },
})    

