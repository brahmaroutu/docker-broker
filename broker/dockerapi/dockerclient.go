package dockerapi

import (
    "bytes"
    "errors"
    "fmt"
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/brokerapi"
    "io/ioutil"
    "net"
    "net/http"
    "net/url"
    "strconv"
    "strings"
    "time"
    "crypto/tls"
    "log"
    "sort"
)

var (
    ErrNotFound = errors.New("Not found")
    ErrConflict = errors.New("Already Exist")
)

type DockerClient struct {
    URL        *url.URL
    HTTPClient *http.Client
    ServiceAgent brokerapi.ServiceAgent
    persister  brokerapi.Persister
    brokerconfig BrokerConfiguration
}

type CFError struct {
    ErrorCode int
    ErrorDesc string
}

type SSLConfig struct {
    ClientCert    []byte
    ClientKey     []byte
    CA            []byte
}

//type Callback func(*Event, ...interface{})

func NewDockerClient(sa brokerapi.ServiceAgent, config BrokerConfiguration) (*DockerClient, error) {
    var urlstr string
    if config.UseSSL(sa.DockerHost) {
        urlstr = "https://" + sa.DockerHost + ":" + strconv.Itoa(sa.DockerPort)
    } else {
        urlstr = "http://" + sa.DockerHost + ":" + strconv.Itoa(sa.DockerPort)
    }
    u, err := url.Parse(urlstr)
    if err != nil {
        log.Println("parsing url ",urlstr," caused ",err)
        return nil, err
    }
    var httpClient *http.Client
    sslConfig := config.GetSSL(sa.DockerHost)
    if config.UseSSL(sa.DockerHost) {
        httpClient = newHTTPsClient(u, sslConfig.ClientCert, sslConfig.ClientKey, sslConfig.CA)
    } else {
        httpClient = newHTTPClient(u)
    }
       config.Persister.Connect()    
    return &DockerClient{u, httpClient, sa, config.Persister,config}, nil
}

func (client *DockerClient) DoRequest(method string, path string, body []byte) ([]byte, error) {
    b := bytes.NewBuffer(body)

    req, err := http.NewRequest(method, client.URL.String()+path, b)
    if err != nil {
        return nil, err
    }
    req.Header.Add("Content-Type", "application/json")
    resp, err := client.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    data, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    if resp.StatusCode == 404 {
        return nil, ErrNotFound
    }
    if resp.StatusCode == 409 {
        return nil, ErrConflict
    }
    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("%s: %s", resp.Status, data)
    }
    return data, nil
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

func newHTTPsClient(u *url.URL, clientCert,clientKey,CA []byte ) *http.Client {
    cert, err := tls.X509KeyPair(clientCert,clientKey)
    if err != nil {
        log.Fatalf("server: loadkeys: %s", err)
    }
    config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
     tr := &http.Transport{
        TLSClientConfig: &config,
    }
    return &http.Client{Transport: tr}
     
}

func (client *DockerClient) Catalog() (brokerapi.Catalog, error) {
    return brokerapi.Catalog{}, brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, "Catalog is not Supported from the client"})
}

func (client *DockerClient) Provision(pr brokerapi.ProvisioningRequest) (string, error) {
    imageName := pr.ServiceId
    log.Println("Looking for image: ", imageName)
    image, err := FindImage(*client, imageName)
    if err != nil {
        return "", brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, err.Error()})
    }
    log.Println("Image:", imageName, "info:", image.Id, "tags:", image.RepoTags)

    imagerepo := strings.Split(image.RepoTags[0], ":")[0]
    config := ContainerConfig{AttachStdout: false, AttachStdin: false, Image: imagerepo, Hostname: pr.InstanceId}
    imagedefinition := client.brokerconfig.GetImageDefinition(imagerepo)
    
    var cId, containername,service_port_str string
    var service_port,host_port int
    var dashurl string

    containername = pr.InstanceId
    if needNewContainer(imagedefinition) {
        log.Println("Creating container:")
        cId, err = config.CreateContainer(*client, pr.InstanceId)
        if err != nil {
            log.Println("creating container caused error ",err)
            if err == ErrConflict {
                //see if this is the container created earlier but failed later, we can reuse the container
                ci, err2 := client.InspectContainer(pr.InstanceId)
                containername = ci.Name
                cId = ci.Id
                err = err2
            }
        }
        ii, err := client.InspectImage(imagerepo)
        if err != nil {
            log.Println("Inspect image caused ", err)
            return "", brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, err.Error()})
        }
        
        hostConfig := &HostConfig{PublishAllPorts: true, NetworkMode: "bridge"}
        pb_min,pb_max,pb,err := client.persister.GetPortBindings("docker_host='"+client.ServiceAgent.DockerHost+"'")
        if err != nil || pb_min == 0 || pb_max == 0 {
            log.Println("Unable to assign a port, port binding not configured(uses default port binding) or error occured ",err)
        } else {
            hostConfig.PortBindings = make(map[string][]PortBinding)
            //normally expected to have one port binding
            for k := range ii.ContainerConfig.ExposedPorts {
                next_port := FindNextNum(pb,pb_min,pb_max)
                pb = append(pb,next_port)
                service_port_str = strings.Split(k,"/")[0]
                service_port,_ = strconv.Atoi(service_port_str)
                host_port = next_port
                log.Println("Mapping ports ",service_port, " to host port ", host_port)
                hostConfig.PortBindings[k] = []PortBinding{PortBinding{HostPort: strconv.Itoa(next_port)}}
            }
            client.persister.WritePortBinding(pb,"docker_host='"+client.ServiceAgent.DockerHost+"'")
        }

        log.Println("Starting container:", containername)
        err = client.StartContainer(cId, hostConfig)
        if err != nil {
            log.Println("StartContainer caused ", err)
            return "", brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, err.Error()})
        }
    } else {
        log.Println("Using existing container",imagedefinition.Containername)
        if (imagedefinition.Numinstances == 0) {
            //Srini we need to get container id from the name by storing it in our DB
            //for now I will just use id for testing
            containername = imagedefinition.Containername
        } else {
            containername = imagedefinition.Containername
        }
        cId = imagedefinition.Containername        

        //TODO we need to get this info        
        service_port = 0
        host_port = 0
    }    
    
    var provision_response map[string] interface{}
    if len(imagedefinition.DashBoardUrl) > 0 || len(imagedefinition.Credentials) > 0 {
        provision_response = imagedefinition.DashBoardUrl
    } else {
        dockerExec := CommandExecutors[client.ServiceAgent.ExecCommand]
        if (dockerExec == nil) {
            return "",errors.New("No Executor specified") 
        }
        dockerExec = dockerExec.Init(client,cId,imagedefinition) 
    
        provision_response,err = dockerExec.(DockerExec).Provision()    
        if err != nil {
            return "", brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, err.Error()})
        }
        
    }

    client.mapServerUrl(provision_response,cId)
    if val,ok := provision_response["dashboard_url"]; ok {
        //dashurl is needed by the defer statement
        dashurl = val.(string)
    } else {
        dashurl,_ = marshalMapToBytes(provision_response)
    }
    

    err = client.persister.AddServiceInstance(imageName, service_port, 
              host_port, dashurl, cId, client.ServiceAgent.DockerHost, 
              containername, imageName, pr, time.Now())

    return dashurl, err
}

func needNewContainer(imagedef *brokerapi.ImageDefinition) bool {
    //Srini check the map that tells how many instances running in each container
    if imagedef.Numinstances == 0 {
        return false
    }
    return true
}

func  (client *DockerClient) shouldRemoveContainer(imagedef *brokerapi.ImageDefinition) bool {
    //Srini check the map that tells how many instances running in each container
    if imagedef.Numinstances == 0 {
        return false
    } 
    return true
}

func (client *DockerClient) mapServerUrl(response map[string]interface{}, cId string) bool {
    if response == nil {
        log.Println("empty response, nothing to replace")
        return false
    }
    ci, err := client.InspectContainer(cId)
    if err != nil {
        log.Println("failed to get container info for ",cId)
        return false
    }
    portbindings := ci.NetworkSettings.Ports

    log.Println("incoming response ",response)
    port_replacement := make(map[string] string)
    var default_port string
    for k, v := range portbindings {
        //replace protocol binding
        k = strings.Split(k, "/")[0]
        for _, pb := range v {
            if (pb.HostIp == client.ServiceAgent.ServiceHost) || (pb.HostIp == "0.0.0.0")  {
                port_replacement["$PORT_"+k]=pb.HostPort
                default_port = pb.HostPort
                break
            }
        }
    }

    for k,v := range response {
        v = strings.Replace(v.(string), "$HOST", client.ServiceAgent.ServiceHost, -1)
        for oldport,newport := range port_replacement {
            v = strings.Replace(v.(string), oldport, newport, -1)
        }
        //replace any other $PORT left out with default port
        v = strings.Replace(v.(string), "$PORT", default_port, -1)
        response[k] = v  
    }
    log.Println("tranformed response ",response)
    
    return true
}

func (client *DockerClient) Deprovision(pr brokerapi.ProvisioningRequest) error {
    cId,imageName := client.persister.GetContainerIdAndImageName(pr.InstanceId)
    if cId == "" {
        return brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeGone, 
                             "Failed to find the Instance in the Database"})
    }

    imagedefinition := client.brokerconfig.GetImageDefinition(imageName)

    defer func() {
        //TODO remove the reserved port binding for this container, we may check multitenancy here
        _,_,pb,_ := client.persister.GetPortBindings("docker_host='"+client.ServiceAgent.DockerHost+"'")
        service_port := client.persister.GetServicePort(pr.InstanceId)
        pb = DeleteItem(pb,service_port)
        client.persister.WritePortBinding(pb,"docker_host='"+client.ServiceAgent.DockerHost+"'")        
        client.persister.DeleteServiceInstance(pr.InstanceId)
    }()
    
    var err error
    if len(imagedefinition.DashBoardUrl) == 0 && len(imagedefinition.Credentials) == 0 {
        dockerExec := CommandExecutors[client.ServiceAgent.ExecCommand]
        if (dockerExec == nil) {
            log.Println("No Executor specified") 
        }
        dockerExec = dockerExec.Init(client,cId,imagedefinition) 
    
        response,err := dockerExec.Deprovision()    
        if err != nil {
            log.Println("Error occurred running deprovision script ",err)
        }
        log.Println("deprovision response is ", response)
    }

    if client.shouldRemoveContainer(imagedefinition) {
        log.Println("Stopping container:", cId)
        err = client.StopContainer(cId, 0)
        if err != nil {
            log.Println("Error stopping container ", err )
            return brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther,
                                                err.Error()})
        }

        log.Println("Removing container:", cId)
        err = client.RemoveContainer(cId)
        if err != nil {
            log.Println("Error Removing container ", err )
            return brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther,
                                                err.Error()})
        }
    }

    return err
}

func (client *DockerClient) Bind(br brokerapi.BindingRequest) (string, brokerapi.Credentials, string, error) {
    cId,imageName := client.persister.GetContainerIdAndImageName(br.InstanceId)
    log.Println("Found container id for ",br.InstanceId," and the value is ",cId,imageName)

    imagedefinition := client.brokerconfig.GetImageDefinition(imageName)

    //write the binding to table
    client.persister.AddServiceBinding(br.InstanceId, br.BindingId, br.AppId, time.Now())
    
    creds := make(brokerapi.Credentials)
    var err error
    if len(imagedefinition.DashBoardUrl) > 0 || len(imagedefinition.Credentials) > 0 {
        for k,v := range imagedefinition.Credentials {
            creds[k] = v
        } 
    } else {
        dockerExec := CommandExecutors[client.ServiceAgent.ExecCommand]
        if (dockerExec == nil) {
            return "",nil,"",errors.New("No Executor specified") 
        }
    
        dockerExec = dockerExec.Init(client,cId,imagedefinition) 
        creds,err = dockerExec.Bind()    
        if err != nil {
            return "",nil,"", brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, err.Error()})
        }
    
        log.Println("bind response is ", creds)
    }
    client.mapServerUrl(creds, cId)

    return "credentials", creds, "unknown", err
}

func (client *DockerClient) Unbind(br brokerapi.BindingRequest) error {
    cId,imageName := client.persister.GetContainerIdAndImageName(br.InstanceId)
    if cId == "" {
        return brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeGone, "Failed to find the Instance in the Database"})
    }

    var err error
    defer func() { err = client.persister.DeleteServiceBinding(br.InstanceId, br.BindingId)
        if err != nil {
            err = brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, "Failed to delete service binding ("+err.Error()+")"})
        }
    }()

    imagedefinition := client.brokerconfig.GetImageDefinition(imageName)
    if len(imagedefinition.DashBoardUrl) > 0 || len(imagedefinition.Credentials) > 0 {
        return nil
    }
    
    serviceurl := client.persister.GetServiceUrl("container_id='" + cId + "'")

    dockerExec := CommandExecutors[client.ServiceAgent.ExecCommand]
    if (dockerExec == nil) {
        return errors.New("No Executor specified") 
    }

    dockerExec = dockerExec.Init(client,cId,imagedefinition) 
    _,err = dockerExec.Unbind(serviceurl)    
    if err != nil {
        return brokerapi.BrokerServiceError(&CFError{brokerapi.ErrCodeOther, err.Error()})
    }
    
    return nil
}

func (cfe *CFError) Code() int {
    return cfe.ErrorCode
}

func (cfe *CFError) Error() string {
    return cfe.ErrorDesc
}

type myint []int

func FindNextNum(inta myint, minnum,maxnum int) int {
    sort.Sort(inta)
    for i := minnum; i < maxnum; i++ {
        if i-minnum>=len(inta) {
            return i
        }
        indx := sort.SearchInts(inta,i)
        if (indx==0 && len(inta)==0) || (inta[indx]!=i) {
            return i
        }
    }
    return -1
}

func DeleteItem(inta myint,val int) []int {
    if len(inta)==0 {
        return inta
    }    
    sort.Sort(inta)
    i := sort.SearchInts(inta,val)
    return append(inta[:i], inta[i+1:]...)
}

    
func (myarr myint) Len() int {
    return len(myarr)
}

func (myarr myint) Swap(i,k int) {
    myarr[i],myarr[k] = myarr[k],myarr[i]
}

func (myarr myint) Less(i,k int) bool {
    return myarr[i]<myarr[k]
}
