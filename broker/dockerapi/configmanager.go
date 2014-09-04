package dockerapi

import (
    "log"
    "github.com/brahmaroutu/docker-broker/broker/brokerapi"
    "io/ioutil"
    "encoding/json"
    "os"
    "strconv"
    "errors"
)

type BrokerConfiguration struct {
    Services     brokerapi.ServiceDefinition
    Persister    brokerapi.Persister
    ListenIP     string
    Port         int
    Dispatcher   string
    BrokerCerts  []brokerapi.BrokerCerts
}


func  NewConfiguration(configFile string) (*BrokerConfiguration, error) {
    if _, err := os.Stat(configFile); os.IsNotExist(err) {
        log.Printf("Config file does not exist '%v': %v\n", configFile, err)
        return nil,err
    }

    file, err := ioutil.ReadFile(configFile)
    if err != nil {
        log.Printf("Cannot read config file '%v': %v\n", configFile, err)
        return nil,err
    }

    cm := BrokerConfiguration{}
    err = json.Unmarshal(file, &cm)
    if err != nil {
        log.Printf("Cannot parse config file '%v': %v\n", configFile, err)
        return nil,err
    }
    
    cm.Persister.Connect()
    
    // if DB has config, we will replace configuration from DB, 
    // else we write this as our first conf to DB.
    if cm.Persister.HasEntry("imageconfigurations","") {
        log.Printf("Using existing config from DB")
        cm.readConfigFromDB()
    } else {
        log.Printf("Using config file to populate config in DB");
        cm.writeConfigToDB()
    }
    return &cm,err    
}

func (cm *BrokerConfiguration) readConfigFromDB() error {
    services,err := cm.Persister.GetServiceConf()
    if err != nil {
        return err
    }
    //we currently expect only on set
    log.Println("Updating services with ",services[0])
    cm.Services = services[0]
        
    brokercerts,err := cm.Persister.GetBrokerCerts()
    if err != nil {
        return err
    }
    log.Println("Updating broker certs")
    cm.BrokerCerts = brokercerts

    return nil
}

func (cm *BrokerConfiguration) writeConfigToDB() error {
    return cm.writeServiceConf()
}


func (cm *BrokerConfiguration) writeServiceConf() error {
    service := cm.Services
    if !cm.Persister.HasEntry("serviceconfigurations","catalog='"+service.Catalog+"'") {
        err := cm.Persister.AddServiceConf(service.User,service.Password,service.Catalog)
        if err != nil {
            log.Println("Error writing ",service," Error:",err)
        }
    }
    id := cm.Persister.GetServiceId(service.Catalog)
    for _,imgdef := range service.Images {
        err := cm.writeImageConf(id,imgdef)
        if err != nil {
            log.Println("Error writing ",imgdef," Error:",err)
            return err
        }
    }
    return nil
}

func marshalMapToBytes(in map[string] interface{}) (string,error) {
    var ret string
    if in != nil && len(in)>0 {
        bytes,err := json.Marshal(in)
        if err != nil {
            return ret,err
        }
        ret = string(bytes)
    }
    return ret,nil
}

func (cm *BrokerConfiguration) MarshalImageMaps(imgdef brokerapi.ImageDefinition) (string,string,error) {
    var dashurl,credentials string
    var err error
    dashurl,err = marshalMapToBytes(imgdef.DashBoardUrl)
    if err != nil {
        log.Println("failed to marshal dashurl ",err)
        return dashurl,credentials,err
    }
    credentials,err = marshalMapToBytes(imgdef.Credentials)
    if err != nil {
        log.Println("failed to marshal credentials ",err)
        return dashurl,credentials,err
    }
    return dashurl,credentials,err
}

func (cm *BrokerConfiguration) writeImageConf(service_id int, imgdef brokerapi.ImageDefinition) error {
    dashurl,credentials,err := cm.MarshalImageMaps(imgdef)
     
    err = cm.Persister.AddImageConf(service_id,imgdef.Name,imgdef.Plan,dashurl, credentials,imgdef.Numinstances,imgdef.Containername)
    return err
}

func (cm *BrokerConfiguration) GetOpts() brokerapi.Options {
    opts := brokerapi.Options{
        Host:     cm.ListenIP,
        Port:     cm.Port,
        Username: cm.Services.User,
        Password: cm.Services.Password,
        Debug:    true,
        LogFile:  "",
        Trace:    false,
        PidFile:  "",
    }
 
    if host := os.Getenv("VCAP_APP_HOST"); host != "" {
        opts.Host = host
    }

    if port := os.Getenv("VCAP_APP_PORT"); port != "" {
        opts.Port,_ = strconv.Atoi(port)
    }
    return opts
}

func (cm *BrokerConfiguration) GetServices() []brokerapi.Service {
    dockerServices := make([]brokerapi.Service, len(cm.Services.Images))
    for i, image := range cm.Services.Images {
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
        log.Printf("  Name: %v Plan: %v", image.Name, image.Plan)
    }
    return dockerServices
}

func (cm *BrokerConfiguration) GetImageDefinition(imagename string) *brokerapi.ImageDefinition {
    if count,_:= cm.Persister.GetCount("imageconfigurations",""); int32(len(cm.Services.Images)) != count {
        cm.readConfigFromDB()
    }
    for _, image := range cm.Services.Images {
        if image.Name == imagename {
            return &image
        }
    }
    return nil    
}

func (cm *BrokerConfiguration) AddOrUpdateImageDefinition(catalog string, img brokerapi.ImageDefinition) error {
    service_id := cm.Persister.GetServiceId(catalog)
    if service_id < 0 {
        return brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, "Catalog is not Supported from the client"})
    }        
    return cm.writeImageConf(service_id, img)
}

func (cm *BrokerConfiguration) GetImageDefinitions(name string) ([]brokerapi.ImageDefinition, error) {
    if len(name)>0 {
        return []brokerapi.ImageDefinition{*cm.GetImageDefinition(name)},nil
    }    
    return cm.Services.Images,nil
    
}

func (cm *BrokerConfiguration) DeleteImageDefinition(catalog,name string) error {
    service_id := cm.Persister.GetServiceId(catalog)
    image := cm.GetImageDefinition(name)
    if len(catalog)==0 || len(name)==0 || image==nil {
        return errors.New("Cannot find image "+name+" to delete")
    }
    
    return cm.Persister.DeleteImageConf(service_id,name,image.Plan)
}

func (cm *BrokerConfiguration) AddOrUpdateCertificates(certs brokerapi.BrokerCerts) error {
    if cm.Persister.HasEntry("brokercertificates","serviceagent='"+certs.Host+"'") {
        return brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, "Certificate Exists for this host"+certs.Host})
    }
    return cm.Persister.AddBrokerCertsConf(certs.Host,certs.ClientCert,certs.ClientKey,certs.CA)
}

func (cm *BrokerConfiguration) GetCertificates(name string) ([]brokerapi.BrokerCerts, error) {
    var err error
    cm.BrokerCerts,err = cm.Persister.GetBrokerCerts()
    if err != nil {
        log.Println("Failed to obtain certs ",err)
    }
    if len(name)>0 {
        for _, certs := range cm.BrokerCerts {
            if certs.Host == name {
                return cm.maskcerts([]brokerapi.BrokerCerts{certs}),nil
            }
        }
    }
    return cm.maskcerts(cm.BrokerCerts),nil
}

func (cm *BrokerConfiguration) maskcerts(certs []brokerapi.BrokerCerts) []brokerapi.BrokerCerts {
    newcerts := make([]brokerapi.BrokerCerts,len(certs))
    for indx,cert := range certs {
        newcert := cert
        newcert.ClientCert = []byte(strconv.Itoa(len(cert.ClientCert)))
        newcert.ClientKey = []byte(strconv.Itoa(len(cert.ClientKey)))
        newcert.CA = []byte(strconv.Itoa(len(cert.CA)))
        newcerts[indx] = newcert
    }
    return newcerts
}

func (cm *BrokerConfiguration) DeleteCertificate(name string) error {
    if len(name)==0 || !cm.Persister.HasEntry("brokercertificates","serviceagent='"+name+"'") {
        return errors.New("Cannot find certificate "+name+" to delete")
    }
    
    return cm.Persister.DeleteBrokerCertsConf(name)
}


func (cm *BrokerConfiguration) UseSSL(host string) bool {
    if !cm.Persister.HasEntry("brokercertificates","serviceagent='"+host+"'") {
        return false;
    }
    cert,_ := cm.GetCertificates(host)
    return (len(cert)==1 && (len(cert[0].ClientCert) > 0 || len(cert[0].ClientKey) > 0)) 
}

func (cm *BrokerConfiguration) GetSSL(host string) *SSLConfig {
    var cert brokerapi.BrokerCerts
    if cm.UseSSL(host) {
        for _,brokercerts := range cm.BrokerCerts {
            if brokercerts.Host == host {
                cert = brokercerts
            }
        }
           return &SSLConfig {ClientCert: cert.ClientCert, ClientKey: cert.ClientKey, CA: cert.CA}
       }
       return nil
}




