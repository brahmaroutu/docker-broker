package dockerapi_test

import (
    "github.com/brahmaroutu/docker-broker/broker/dockerapi"
    "github.com/brahmaroutu/docker-broker/broker/brokerapi"
    "github.com/brahmaroutu/docker-broker/broker/testhelpers"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "fmt"
    "time"
    "errors"
    )

var _ = Describe("Simpledispatcher", func() {
    var config    dockerapi.BrokerConfiguration
    var persister brokerapi.Persister
    var dispatcher brokerapi.DispatcherInterface
    var cm dockerapi.BrokerConfiguration
    var err error
    JustBeforeEach(func() {
        config = testnet.BrokerConfiguration(); 
        dispatcher = testnet.SimpleDispatcher()
        persister = config.Persister
        cm = testnet.BrokerConfiguration()
    })
    
    Describe("brokertest test simpledispatcher", func() {
        BeforeEach(func() {        
            testnet.SetupSQL(persister)
        })
        AfterEach(func() {
            testnet.CleanupSQL(persister)
        })

        It("cannot create dispatcher when no agents are registered", func() {
            var newbrokerservice brokerapi.BrokerService
            newbrokerservice, err = dispatcher.NewBrokerService()
            Expect(err).To(Equal(errors.New("no agents available")))
            Expect(newbrokerservice).Should(BeNil())    
        })


        It("cannot create dispatcher when last ping form agent is old", func() {
            sa := testnet.NewServiceAgent()

            persister.AddorUpdateServiceAgent(sa)            
            result,errs := persister.Db.Exec("update serviceagents set last_ping=?",time.Now().Add(-300*time.Minute))
            fmt.Println(errs)
            if err == nil {
                i,_ := result.RowsAffected()
                fmt.Println(i)
            }
            
            var newbrokerservice brokerapi.BrokerService
            newbrokerservice, err = dispatcher.NewBrokerService()
            Expect(err).To(Equal(errors.New("no agents available")))
            Expect(newbrokerservice).Should(BeNil())    
        })


        It("create dispatcher that is current", func() {
            sa := testnet.NewServiceAgent()

            persister.AddorUpdateServiceAgent(sa)            
            persister.Db.Exec("update serviceagents set last_ping=?",time.Now().Add(-300*time.Minute))

            sa2 := testnet.NewServiceAgent()
            sa2.DockerHost = "localhost2";
            sa2.DockerPort = 555;

            persister.AddorUpdateServiceAgent(sa2)            

            var newbrokerservice brokerapi.BrokerService
            newbrokerservice, err = dispatcher.NewBrokerService()
            Expect(err).To(BeNil())
            Expect(newbrokerservice).ShouldNot(BeNil())
        })
        
    })
})
