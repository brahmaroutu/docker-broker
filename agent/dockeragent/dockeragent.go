package dockeragent

import (
    "bytes"
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "net/url"
    "strconv"
    "time"
    "encoding/json"
)

var (
    ErrNotFound = errors.New("Not found")
)

type DockerAgent struct {
    URL           *url.URL
    HTTPClient    *http.Client
    Broker        DockerBroker  
    Serviceagent  ServiceAgent
    //may want to keep the list of containers, services run, when last service deployed, running since?
}


func NewDockerAgent(broker DockerBroker, sa ServiceAgent) (*DockerAgent, error) {
    urlstr := "http://"+broker.Host
    
    if broker.Port > 0 {
        urlstr = urlstr +":"+strconv.Itoa(broker.Port)
    }
    
    u, err := url.Parse(urlstr)
    if err != nil {
        return nil, err
    }
    httpClient := newHTTPClient(u)
    return &DockerAgent{u, httpClient,broker,sa}, nil
}

func (client *DockerAgent) DoRequest(method string, path string, body []byte) ([]byte, error) {
    b := bytes.NewBuffer(body)
    req, err := http.NewRequest(method, client.URL.String()+path, b)
    if err != nil {
        return nil, err
    }
//    req.Header.Add("Content-Type", "application/json")
    req.Header.Add("X-Broker-Api-Version","1.1")
    req.SetBasicAuth(client.Broker.User,client.Broker.Password)
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

func (client *DockerAgent) Register(clientCertFile,clientKeyFile,caFile string) error {    
    client.UploadCerts(clientCertFile,clientKeyFile,caFile)
    go client.Ping(time.Duration(client.Serviceagent.KeepAlive) * time.Second)
    return nil
} 

func (client *DockerAgent) UploadCerts(clientCertFile,clientKeyFile,caFile string) {
    var clientCert,clientKey,CA []byte
    if len(clientCertFile)>0 {
        log.Println("Uploading client cert file")
        clientCert = ReadFile(clientCertFile)        
    }
    if len(clientKeyFile)>0 {
        log.Println("Uploading client key file")
        clientKey = ReadFile(clientKeyFile)        
    }
    if len(caFile)>0 {
        log.Println("Uploading ca cert file")
        CA = ReadFile(caFile)        
    }
    if len(clientCert) == 0 && len(clientKey) == 0 {
        return
    }
    certs := BrokerCerts{Host:client.Serviceagent.DockerHost,ClientCert:clientCert,ClientKey:clientKey,CA:CA}

    u, err := url.Parse("/certificate/"+client.Serviceagent.DockerHost)
    if err != nil {
        log.Println( "Unable to load certificates - parsing issue: ", err)
        return
    }

    b,err := json.Marshal(certs)
    if err != nil {
        log.Println("Unable to load certificates - failed to marshal: ", 
                    certs)
        return
    }
    _,err = client.DoRequest("PUT", u.String(), b)
    if err != nil {
        log.Println("Error on Put ",err)
    }    
}

func (client *DockerAgent) Ping( delay time.Duration ) {
    log.Println("Entering ping/registration loop")

    u, err := url.Parse("/ping")
    if err != nil {
        log.Println( "Stopping ping loop - parsing issue: ", err)
        return
    }

    b,err := json.Marshal(client.Serviceagent)
    if err != nil {
        log.Println("Stopping ping loop - failed to marshal: ", 
                    client.Serviceagent)
        return
    }
    log.Println( "Registration data: ", string(b) )

    var prevErr error = errors.New( "" )

    for {
        _,err = client.DoRequest("POST", u.String(), b)
        if err != nil && (prevErr == nil || err.Error()!=prevErr.Error()) {
            log.Println("Error connecting to broker(will keep trying)" )
            log.Println("Error:", err )
        }
        if err == nil && prevErr != nil {
            log.Println("Connected to Broker successfully")
        }
        prevErr = err 
        time.Sleep( delay )
    }
    log.Println("Exiting ping loop")
}

func (client *DockerAgent) GetPerfFactor() float32  {
    return 1.22
}

func ReadFile(filename string) []byte {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        fmt.Println("Error reading file ",filename," and the error is ",err)
    }
    return data
}
