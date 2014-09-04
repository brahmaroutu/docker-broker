package dockerapi

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"
)

type ImageInfo struct {
    RepoTags    []string
    Id          string
    Parentid    string
    Created     uint64
    Size        uint64
    Virtualsize uint64
}


type ImageConfig struct {
    ID              string          
    Parent          string          
    Comment         string          
    Created         time.Time       
    Container       string          
    ContainerConfig ImageRuntimeConfig
    DockerVersion   string            
    Author          string            
    Config          *ImageRuntimeConfig 
    Architecture    string          
    OS              string          
    Size            int64
}

type ImageRuntimeConfig struct {
    Hostname        string
    Domainname      string
    User            string
    Memory          int64  // Memory limit (in bytes)
    MemorySwap      int64  // Total memory usage (memory + swap); set `-1' to disable swap
    CpuShares       int64  // CPU shares (relative weight vs. other containers)
    Cpuset          string // Cpuset 0-2, 0,1
    AttachStdin     bool
    AttachStdout    bool
    AttachStderr    bool
    PortSpecs       []string // Deprecated - Can be in the format of 8080/tcp
    ExposedPorts    map[string]struct{}
    Tty             bool // Attach standard streams to a tty, including stdin if it is not closed.
    OpenStdin       bool // Open stdin
    StdinOnce       bool // If true, close stdin after the 1 attached client disconnects.
    Env             []string
    Cmd             []string
    Image           string // Name of the image as it was passed by the operator (eg. could be symbolic)
    Volumes         map[string]struct{}
    WorkingDir      string
    Entrypoint      []string
    NetworkDisabled bool
    OnBuild         []string
}


func ListAll(docker DockerClient) ([]ImageInfo, error) {
    data, err := docker.DoRequest("GET", "/images/json?all=0", nil)
    if err != nil {
        fmt.Println("ListAll images caused ", err)
        return nil, err
    }
    imageinfo := []ImageInfo{}
    err = json.Unmarshal(data, &imageinfo)
    if err != nil {
        fmt.Println("ListAll failed to unmarshal: ",string(data), " Err: ", err)
        return nil, err
    }
    return imageinfo, nil
}

func (imageinfo *ImageInfo) GetRepoTags() []string {
    return imageinfo.RepoTags
}

func FindImage(docker DockerClient, tag string) (ImageInfo, error) {
    images, err := ListAll(docker)
    if err != nil {
        return ImageInfo{}, err
    }
    //Srini looking for image, we need to match to a level. May be we split on ":"
    for _, image := range images {
        for _, repotag := range image.RepoTags {
            if strings.Split(repotag, ":")[0] == tag || repotag == tag {
                return image, nil
            }
        }
    }
    return ImageInfo{}, fmt.Errorf("Cannot find image")
}

func (docker *DockerClient) InspectImage(imageName string) (ImageConfig, error) {
    config := ImageConfig{}
    config.ContainerConfig = ImageRuntimeConfig{}
    config.Config = &ImageRuntimeConfig{}
    data, err := docker.DoRequest("GET", "/images/"+imageName+"/json", nil)

    if err != nil {
        fmt.Println("InspectImage caused ", err)
        return config, err
    }
    err = json.Unmarshal(data, &config)

    if err != nil {
        fmt.Println("InspectImage failed to unmarshal: ",string(data), " Err: ", err)
        return config, err
    }
    return config, nil
}
