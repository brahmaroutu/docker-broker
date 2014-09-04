package dockerapi

import (
    "encoding/json"
    "fmt"
    "strings"
    "unicode/utf8"
    "errors"
)

type ContainerConfig struct {
    Hostname        string
    Domainname      string
    User            string
    Memory          int
    MemorySwap      int
    CpuShares       int
    Cpuset          string
    AttachStdin     bool
    AttachStdout    bool
    AttachStderr    bool
    PortSpecs       []string
    ExposedPorts    map[string]struct{}
    Tty             bool
    OpenStdin       bool
    StdinOnce       bool
    Env             []string
    Cmd             []string
    Image           string
    Volumes         map[string]struct{}
    WorkingDir      string
    Entrypoint      []string
    NetworkDisabled bool
    OnBuild         []string
}

type ContainerInfo struct {
    Id      string
    Created string
    Path    string
    Name    string
    Args    []string
    Config  *ContainerConfig
    State   struct {
        Running   bool
        Pid       int
        ExitCode  int
        StartedAt string
        Ghost     bool
    }
    Image           string
    NetworkSettings struct {
        IpAddress   string
        IpPrefixLen int
        Gateway     string
        Bridge      string
        Ports       map[string][]PortBinding
    }
    SysInitPath    string
    ResolvConfPath string
    Volumes        map[string]string
    HostConfig     HostConfig
}

type Port struct {
    PrivatePort int
    PublicPort  int
    Type        string
}

type RestartPolicy struct {
    Name string
    MaxRetry int `json:"MaximumRetryCount"`
}

func RestartOnFailure(maxRetry int) RestartPolicy {
    return RestartPolicy{Name: "on-failure", MaxRetry: maxRetry}
}


type HostConfig struct {
    Binds           []string
    ContainerIDFile string
    LxcConf         []map[string]string
    Privileged      bool
    PortBindings    map[string][]PortBinding
    Links           []string
    PublishAllPorts bool
    Dns             []string
    DnsSearch       []string
    VolumesFrom     []string
    NetworkMode     string
    RestartPolicy RestartPolicy
}

type PortBinding struct {
    HostIp   string
    HostPort string
}

type Container struct {
    Id         string
    Names      []string
    Image      string
    Command    string
    Created    int
    Status     string
    Ports      []Port
    SizeRw     int
    SizeRootFs int
}

type RespContainersCreate struct {
    Id       string
    Warnings []string
}

func (config *ContainerConfig) CreateContainer(docker DockerClient, name string) (string, error) {
    data, err := json.Marshal(config)
    if err != nil {
        return "", err
    }
    uri := "/containers/create?name="+name

    data, err = docker.DoRequest("POST", uri, data)
    if err != nil {
        return "", err
    }
    result := &RespContainersCreate{}
    err = json.Unmarshal(data, result)
    if err != nil {
        return "", err
    }
    return result.Id, nil
}

func (client *DockerClient) StartContainer(id string, config *HostConfig) error {
    data, err := json.Marshal(config)
    if err != nil {
        return err
    }
    uri := fmt.Sprintf("/containers/%s/start", id)
    _, err = client.DoRequest("POST", uri, data)
    return err
}

func (client *DockerClient) StopContainer(id string, timeout int) error {
    uri := fmt.Sprintf("/containers/%s/stop?t=%d", id, timeout)
    _, err := client.DoRequest("POST", uri, nil)
    return err
}

func (client *DockerClient) RemoveContainer(id string) error {
    _, err := client.DoRequest("DELETE", fmt.Sprintf("/containers/%s", id), nil)
    return err
}

func (client *DockerClient) InspectContainer(id string) (ContainerInfo,error) {
    data, err := client.DoRequest("GET", fmt.Sprintf("/containers/%s/json", id), nil)
    var ci ContainerInfo      
    json.Unmarshal(data,&ci)
    return ci,err
}

type CopyFromContainerOptions struct {
    Resource string
}

func (client *DockerClient) ReadFile(id, filename string) (map[string]interface{}, error) {
    opts := CopyFromContainerOptions{Resource: filename}
    data, err := json.Marshal(opts)
    body, err := client.DoRequest("POST", fmt.Sprintf("/containers/%s/copy", id), data)
    if err != nil  {
        return map[string]interface{}{}, err
    }
    if (len(body) <= 0) || (!strings.Contains(string(body),"{")) {
        return map[string]interface{}{}, errors.New("Incorrect response")
    }
    response := strings.Split(string(body), "{")[1]
    lines := strings.Split(response, "\n")
    newresponse := strings.Join(lines[0:], "")
    newresponse = "{" + strings.Split(string(newresponse), "}")[0] + "}"
    strlen := utf8.RuneCountInString(newresponse)
    response_marshalled := []byte(newresponse)
    var objMap map[string]interface{}
    err = json.Unmarshal(response_marshalled[:strlen], &objMap)

    return objMap, err
}
