package dockeragent

import (
    "time"
)

type ServiceAgent struct {
    ServiceHost  string
    DockerHost   string
    DockerPort   int
    StartedAt    time.Time
    IsActive     bool
    PerfFactor   float32
    ExecCommand  string
    ExecArgs     string
    KeepAlive    int //time in secs
    Portbind_min int
    Portbind_max int
}

// Info used to talk to the Service Broker
type DockerBroker struct {
    Host     string
    Port     int
    User     string
    Password string
}

type BrokerCerts struct {
    Host       string
    ClientCert []byte
    ClientKey  []byte
    CA         []byte
}
