// Copyright 2014, The cf-service-broker Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.

package brokerapi

import (
    "time"
    )
    
// The BrokerService defines the internal API used by the broker's HTTP endpoints.
type BrokerService interface {

    // Exposes the catalog of services managed by this broker.
    // Returns the exposed catalog.
    Catalog() (Catalog, error)

    // Creates a service instance of a specified service and plan.
    // Returns the optional management URL.
    Provision(ProvisioningRequest) (string, error)

    // Removes created service instance.
    Deprovision(ProvisioningRequest) error

    // Binds to specified service instance.
    // Returns  credentials necessary to establish connection to this
    // service instance as well as optional syslog drain URL.
    Bind(BindingRequest) (string, Credentials, string, error)

    // Removes created binding.
    Unbind(BindingRequest) error
    
        
    //api for broker management
    //
    
    //agent node uses it to register
    //Register(ServiceAgent) (BrokerService,error)
    
}

type AgentManagerInterface interface {
    Ping(ServiceAgent) error

    Catalog() (Catalog, error)
    
    GetServiceAgent(instanceid string) (BrokerService, error)
    
    AddImage(string,ImageDefinition) error
    GetImage(string,string) ([]ImageDefinition,error)
    DeleteImage(string,string) error
    
    AddCerts(BrokerCerts) error
    GetCerts(string) ([]BrokerCerts,error)
    DeleteCerts(string) error

}

type DispatcherInterface interface {
    //algorithm implementation to find best ServiceAgent for provisioning    
    NewBrokerService() (BrokerService, error)
}

const (
    // Raised by Broker Service if service instance or service instance binding already exists
    ErrCodeConflict = 10
    // Raised by Broker Service if service instance or service instance binding cannot be found
    ErrCodeGone = 20
    // Raised by Broker Service for any other issues
    ErrCodeOther = 99
)

type BrokerServiceError interface {
    Code() int
    Error() string
}

// See http://docs.cloudfoundry.com/docs/running/architecture/services/api.html#provisioning
type ProvisioningRequest struct {
    InstanceId string `json:"-"`
    ServiceId  string `json:"service_id"`
    PlanId     string `json:"plan_id"`
    OrgId      string `json:"organization_guid"`
    SpaceId    string `json:"space_guid"`
}

// See http://docs.cloudfoundry.com/docs/running/architecture/services/api.html#binding
type BindingRequest struct {
    InstanceId string `json:"-"`
    BindingId  string `json:"-"`
    ServiceId  string `json:"service_id"`
    PlanId     string `json:"plan_id"`
    AppId      string `json:"app_guid"`
}

type Credentials map[string]interface{}

// See http://docs.cloudfoundry.com/docs/running/architecture/services/api.html#catalog-mgmt
type Catalog struct {
    Services []Service `json:"services"`
}

// See http://docs.cloudfoundry.com/docs/running/architecture/services/api.html#catalog-mgmt
type Service struct {
    Id          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Bindable    bool                   `json:"bindable"`
    Tags        []string               `json:"tags,omitempty"`
    Requires    []string               `json:"requires,omitempty"`
    Plans       []Plan                 `json:"plans"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// See http://docs.cloudfoundry.com/docs/running/architecture/services/api.html#catalog-mgmt
type Plan struct {
    Id          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Other types
type BrokerError struct {
    Description string `json:"description"`
}

type DockerInfo struct {
    Containers     int
    Images         int
    Debug          int
    NFd            int
    NGoroutines    int
    MemoryLimit    int
    SwapLimit      int
    IPv4Forwarding int
}

type Version struct {
    Version   string
    GitCommit string
    GoVersion string
}

type ImageDefinition struct {
    Name          string
    Plan          string
    DashBoardUrl  map[string] interface{}
    Credentials   map[string] interface{}
    Numinstances  int
    Containername string
}

type ServiceDefinition struct {
    User     string
    Password string
    Catalog  string
    Images   []ImageDefinition
}

type ServiceAgent struct {
    ServiceHost  string
    DockerHost   string
    DockerPort   int
    LastPing     time.Time
    IsActive     bool
    PerfFactor   float32
    KeepAlive    int
    ExecCommand  string
    ExecArgs     string
    Portbind_min int
    Portbind_max int
    Portbindings []uint8
}

type BrokerCerts struct {
    Host       string
    ClientCert []byte
    ClientKey  []byte
    CA         []byte
}


