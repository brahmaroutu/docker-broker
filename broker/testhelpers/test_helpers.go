package testnet

import (
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/brokerapi"    
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/dockerapi"

    . "github.com/onsi/gomega"

    "net/url"
    "net/http"
    "net/http/httptest"
    "strings"
    "strconv"
    "time"
)

func NewTestRequest(request TestRequest) TestRequest {
    request.Header = http.Header{
        "content-type":  {"application/json"},
    }
    return request
}

func NewBrokerServiceWithMultipleRequests(serviceagent brokerapi.ServiceAgent, persister brokerapi.Persister, requests []TestRequest) (*dockerapi.DockerClient, brokerapi.ServiceAgent, *httptest.Server, *TestHandler) {
    ts, handler := NewServer(requests)
    u, err := url.Parse(ts.URL)
    if err != nil {
        Ω(err).ShouldNot(HaveOccurred())
    }
    urlinfo := strings.Split(u.Host,":")
    serviceagent.DockerHost = urlinfo[0]
    serviceagent.DockerPort,_ = strconv.Atoi(urlinfo[1])
            
    brokerservice, err := dockerapi.NewDockerClient(serviceagent, BrokerConfiguration())
    return brokerservice,serviceagent,ts,handler
}

func BrokerConfiguration() dockerapi.BrokerConfiguration {
    persister := NewPersister()
    persister.Connect()
    
    
    id := []brokerapi.ImageDefinition {
        mysqlimg,ubuntuimg, 
    }
    
    sd := brokerapi.ServiceDefinition {
        User: "admin",
          Password: "admin",
          Catalog: "My Docker Catalog",
          Images: id,
    }      
      
    cm := dockerapi.BrokerConfiguration {
        Services:     sd,
        Persister:    persister,
        ListenIP: "127.0.0.1",
        Port: 1234,
    }

    persister.AddServiceConf(sd.User,sd.Password,sd.Catalog)
    service_id := persister.GetServiceId(sd.Catalog)
    for _,imgdef := range sd.Images {
        dashurl,credentials,_ := cm.MarshalImageMaps(imgdef) 
        persister.AddImageConf(service_id,imgdef.Name,imgdef.Plan,dashurl, credentials,imgdef.Numinstances,imgdef.Containername)
    }
    
    return cm
    
}

func SimpleDispatcher() brokerapi.DispatcherInterface {
    return dockerapi.NewSimpleDispatcher(BrokerConfiguration())

}

func NewBrokerService(serviceagent brokerapi.ServiceAgent, persister brokerapi.Persister, req TestRequest) (*dockerapi.DockerClient, brokerapi.ServiceAgent, *httptest.Server, *TestHandler) {
    requests := []TestRequest{req}
    return NewBrokerServiceWithMultipleRequests(serviceagent, persister,requests)
}

func NewServiceAgent() brokerapi.ServiceAgent {
    return brokerapi.ServiceAgent {
        DockerHost: "localhost",
        DockerPort: 1234,
        ServiceHost: "localhost",
        LastPing: time.Now(),
        IsActive: true,
        PerfFactor: 1.0,
        KeepAlive: 1,
    }
}

func NewPersister() brokerapi.Persister {
    return brokerapi.Persister {
        Driver: "sqlite3",
          Host: "localhost",
          Port: 3306,
          User: "docker",
          Password: "docker",
          Database: "broker_testdb",
     }
}

func NewDockerServices() []brokerapi.Service {
    images := []brokerapi.ImageDefinition { mysqlimg,ubuntuimg }
    dockerServices := make([]brokerapi.Service, len(images))
    for i, image := range images {
        dockerServices[i] = brokerapi.Service{
        Id:          image.Name,
        Name:        image.Name,
        Description: image.Name + " docker service",
        Bindable:    true,
        Tags:        []string{"docker"},
        Plans: []brokerapi.Plan{
            brokerapi.Plan{
                Id:          image.Name + "_" + image.Plan,
                Name:        image.Plan,
                Description: "Service plan",
            },
        },
        Metadata: map[string]interface{}{
            "displayName":         "docker image",
            "imageUrl":            nil,
            "longDescription":     "Docker container with chosen functionality",
            "providerDisplayName": "docker",
            "documentationUrl":    nil,
            "supportUrl":          nil},
        }
    }
    return dockerServices
}

var mysqlimg = brokerapi.ImageDefinition {
            Name: "mysql",
            Plan: "100", 
            DashBoardUrl: map[string]interface{} {
                    "dashboard_url":"mysql://fakehost:1234",
                    },
            Credentials: map[string]interface{} {
                    "database":"fakeDB",
                    "url":"mysql://fakehost:1234",
                    "uri":"mysql://fakehost:1234",
                    "host":"fakeHost",
                    "port":"1234",
                    "user":"user31472",
                    "username":"fakeUser",
                    "password":"fakePassword",
                    },
            Numinstances: 1,
            Containername: "820b8fbf18c6",
          }
var ubuntuimg = brokerapi.ImageDefinition {
            Name: "ubuntu",
            Plan: "100",
           }
           
func SetupSQL(persister brokerapi.Persister) {

}

func CleanupSQL(persister brokerapi.Persister) {
    persister.Connect()
    persister.Db.Exec("delete from serviceagents")
    persister.Db.Exec("delete from servicebindings")
    persister.Db.Exec("delete from serviceinstances")
    persister.Db.Exec("delete from brokerconfigurations")
    persister.Db.Exec("delete from imageconfigurations")
    persister.Db.Exec("delete from serviceconfigurations")
    persister.Db.Exec("delete from brokercertificates")
}

var Exec_CommandResponse = map[string] map[string] interface{} {
        "Provision": { "host": "fakeHost", "port": "1234", "user": "fakeUser", "password": "fakePassword","database": "fakeDB", "url": "mysql://fakehost:1234" },
        "Bind": { "host": "fakeHost", "port": "1234", "user": "fakeUser", "password": "fakePassword","database": "fakeDB", "url": "mysql://fakehost:1234" },
        "Deprovision": {"remove":true},
        }


var Provision_ListAllImagesRequest = NewTestRequest(TestRequest{
    Method:  "GET",
    Path:    "/images/json",
    Response: TestResponse{
        Status: http.StatusOK,
        Body: `[
          {
            "RepoTags": [
                   "mysql:5.6.19",
                   "mysql:precise",
                   "mysql:latest"
             ],
             "Id": "myFakeImageId1",
             "Created": 1365714795,
             "Size": 131506275,
             "VirtualSize": 131506275
          },
          {
            "RepoTags": [
                   "ubuntu:12.10",
                   "ubuntu:quantal"
             ],
             "ParentId": "myFakeParentId2",
             "Id": "myFakeImageId2",
             "Created": 1364102658,
             "Size": 24653,
             "VirtualSize": 180116135
          }
        ]`,
    },
})        

var Provision_CreateContainerRequest = NewTestRequest(TestRequest{
    Method:  "POST",
    Path:    "/containers/create?name=myFakeInstance",
    Matcher: RequestBodyMatcher(`{"User": "","Memory": 0,"PortSpecs": null,"StdinOnce": false,
                    "Image": "mysql","Domainname": "","Cpuset": "","AttachStderr": false,"ExposedPorts": null,"Tty": false,"Cmd": null,"MemorySwap": 0,"CpuShares": 0,"AttachStdin": false,"OpenStdin": false,"WorkingDir": "","NetworkDisabled": false,"OnBuild": null,
                    "Hostname": "myFakeInstance","AttachStdout": false,"Env": null,"Volumes": null,"Entrypoint": null}`),
    Response: TestResponse{
        Status: http.StatusOK,
        Body : `{
            "Id":"myFakeInstance",
            "Warnings":[]
        }`,
    },

})    

var Provision_StartContainerRequest = NewTestRequest(TestRequest{
    Method:  "POST",
    Path:    "/containers/myFakeInstance/start",
    Matcher: RequestBodyMatcher(`{"ContainerIDFile": "","Privileged": false,"PublishAllPorts": true,"NetworkMode": "bridge","Binds": null,"PortBindings": null,"Links": null,"Dns": null,"DnsSearch": null,"VolumesFrom": null,"LxcConf": null,"RestartPolicy": {"Name": "","MaximumRetryCount": 0}}`),
    Response: TestResponse{
        Status: http.StatusOK,
    },
})    


var Provision_InspectContainerRequest = NewTestRequest(TestRequest{
    Method:  "GET",
    Path:    "/containers/myFakeInstance/json",
    Response: TestResponse{
        Status: http.StatusOK,
        Body: `{
                 "Id": "myFakeInstance",
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
                             "1234/tcp": [
                             {
                                  "HostIp": "192.999.888.777",
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

var Provision_InspectImageRequest = NewTestRequest(TestRequest{
    Method:  "GET",
    Path:    "/images/mysql/json",
    Response: TestResponse{
        Status: http.StatusOK,
        Body: `{
            "Architecture": "amd64",
            "Author": "",
            "Comment": "",
            "Config": {
                "AttachStderr": false,
                "AttachStdin": false,
                "AttachStdout": false,
                "Cmd": [
                    "/bin/sh",
                    "-c",
                    "/run.sh"
                ],
                "CpuShares": 0,
                "Cpuset": "",
                "Domainname": "",
                "Entrypoint": null,
                "Env": [
                    "HOME=/",
                    "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
                ],
                "ExposedPorts": {
                    "3306/tcp": {}
                },
                "Hostname": "347a3d6eaa91",
                "Image": "aa19bf58589d33f0fb78ab72a240290c32e737615c97de4f3dfcec23d4564753",
                "Memory": 0,
                "MemorySwap": 0,
                "NetworkDisabled": false,
                "OnBuild": [],
                "OpenStdin": false,
                "PortSpecs": null,
                "StdinOnce": false,
                "Tty": false,
                "User": "",
                "Volumes": null,
                "WorkingDir": ""
            },
            "Container": "b0bb5fc46d0b07de83327da6dad18add7bc079e9908e640cfaa1d2ea924ed5ad",
            "ContainerConfig": {
                "AttachStderr": false,
                "AttachStdin": false,
                "AttachStdout": false,
                "Cmd": [
                    "/bin/sh",
                    "-c",
                    "#(nop) CMD [/bin/sh -c /run.sh]"
                ],
                "CpuShares": 0,
                "Cpuset": "",
                "Domainname": "",
                "Entrypoint": null,
                "Env": [
                    "HOME=/",
                    "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
                ],
                "ExposedPorts": {
                    "3306/tcp": {}
                },
                "Hostname": "347a3d6eaa91",
                "Image": "aa19bf58589d33f0fb78ab72a240290c32e737615c97de4f3dfcec23d4564753",
                "Memory": 0,
                "MemorySwap": 0,
                "NetworkDisabled": false,
                "OnBuild": [],
                "OpenStdin": false,
                "PortSpecs": null,
                "StdinOnce": false,
                "Tty": false,
                "User": "",
                "Volumes": null,
                "WorkingDir": ""
            },
            "Created": "2014-09-03T09:16:09.7706864Z",
            "DockerVersion": "1.0.0",
            "Id": "10579d0a97b065dd4ea7a6fa7ba9f68372c16e116fc8859146eb5a06be8bd440",
            "Os": "linux",
            "Parent": "aa19bf58589d33f0fb78ab72a240290c32e737615c97de4f3dfcec23d4564753",
            "Size": 0
        }`,
    },
})   

var Deprovision_StopContainerRequest = NewTestRequest(TestRequest{
    Method:  "POST",
    Path:    "/containers/myFakeInstance/stop?t=0",
    Response: TestResponse{
        Status: http.StatusOK,
    },
})    

var Deprovision_RemoveContainerRequest = NewTestRequest(TestRequest{
    Method:  "DELETE",
    Path:    "/containers/myFakeInstance",
    Response: TestResponse{
        Status: http.StatusOK,
    },
})    