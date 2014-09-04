package dockerapi_test

import (
    "github.com/brahmaroutu/docker-broker/broker/dockerapi"
    "github.com/brahmaroutu/docker-broker/broker/brokerapi"
    "github.com/brahmaroutu/docker-broker/broker/testhelpers"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "net/http"
    "net/http/httptest"
    
)

var _ = Describe("Images", func() {
    var ImageInfo1, ImageInfo2 dockerapi.ImageInfo
    var brokerservice *dockerapi.DockerClient
    var serviceagent brokerapi.ServiceAgent
    var persister brokerapi.Persister
    var ts *httptest.Server
    var handler *testnet.TestHandler   
    var err error
    JustBeforeEach(func() {
        ImageInfo1 = dockerapi.ImageInfo {
            Id:"myFakeImageId1",
             RepoTags:[]string{"ubuntu:12.04",
                       "ubuntu:precise",
                       "ubuntu:latest",
            },
              Created: 1365714795,
              Size: 131506275,
               Virtualsize: 131506275,
        }    
        ImageInfo2 = dockerapi.ImageInfo {
            Id:"myFakeImageId2",
             Parentid:"myFakeParentId2",
             RepoTags:[]string{"ubuntu:12.10",
                       "ubuntu:quantal",
            },
            Created: 1364102658,
            Size: 24653,
            Virtualsize: 180116135,
        }   

        serviceagent = testnet.NewServiceAgent()        
        persister = testnet.NewPersister()
    
        brokerservice,serviceagent,ts,handler = testnet.NewBrokerService(serviceagent,persister, listAllImageRequest)    
    })

    AfterEach( func() {
        defer ts.Close()    
    })
    
    Describe("brokertest test images", func() {
        It("should listall images", func() {
            var imageInfos []dockerapi.ImageInfo 
            imageInfos,err = dockerapi.ListAll(*brokerservice)
            Expect(err).To(BeNil())
            Expect(imageInfos).To(Equal([]dockerapi.ImageInfo{ImageInfo1,ImageInfo2}))
        })

        It("should find image", func() {
            var imageInfo dockerapi.ImageInfo 
            imageInfo,err = dockerapi.FindImage(*brokerservice, "ubuntu:12.04")
            Expect(err).To(BeNil())
            Expect(imageInfo).To(Equal(ImageInfo1))
        })

        It("should get repo tags from image", func() {
            repotags := ImageInfo1.GetRepoTags()
            Expect(repotags).To(Equal(ImageInfo1.RepoTags))
        })

    })
    
})


var listAllImageRequest = testnet.NewTestRequest(testnet.TestRequest{
    Method:  "GET",
    Path:    "/images/json",
    Response: testnet.TestResponse{
        Status: http.StatusOK,
        Body: `[
          {
            "RepoTags": [
                   "ubuntu:12.04",
                   "ubuntu:precise",
                   "ubuntu:latest"
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
