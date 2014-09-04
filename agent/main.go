package main

import (
    "encoding/json"
    "io/ioutil"
    "log"
    "os"
    "os/signal"
     "flag"
    "github.com/brahmaroutu/docker-broker/agent/dockeragent"
    )
    

    
type AgentConfiguration struct {
    Serviceagent dockeragent.ServiceAgent
    Brokerservers []dockeragent.DockerBroker
}
    
func main() {
    /* Usage: agent [ -config file ] */

    var configFile string
    var clientCertFile,clientKeyFile,caFile string  

    flag.StringVar( &configFile, "config", "agent.config","Location of configuration file" )
    flag.StringVar( &clientCertFile, "clientcert", "","Client Certificate File")
    flag.StringVar( &clientKeyFile, "clientkey", "","ClientKey File")
    flag.StringVar( &caFile, "cacert", "","CA Certificate File")    

    flag.Parse()

    log.Println( "ConfigFile:", configFile )
    file, e := ioutil.ReadFile(configFile)
    if e != nil {
        log.Printf("Cannot read config file '%v'; %v\n", configFile, e)
        os.Exit(1)
    }

    config := AgentConfiguration{}
    err := json.Unmarshal(file, &config)
    if err != nil {
      log.Println( "Error parsing config file(", configFile, "): ", err )
      os.Exit(1)
    }
   
    brokerServers := make([]*dockeragent.DockerAgent,len(config.Brokerservers))
    for i,broker := range config.Brokerservers {
        brokerServers[i],_ = dockeragent.NewDockerAgent(broker,config.Serviceagent)
        log.Println("args to register ",clientCertFile,clientKeyFile,caFile)
        brokerServers[i].Register(clientCertFile,clientKeyFile,caFile)
    }
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
                        
    for {
        select {
         case sig := <-sigCh:
            var _ = sig
            log.Println("Agent shutdown gracefully")
            return
        }
    }
    
}
