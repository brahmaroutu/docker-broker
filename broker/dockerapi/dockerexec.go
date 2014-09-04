package dockerapi

import (
    "github.rtp.raleigh.ibm.com/bluemix/docker-broker/broker/brokerapi"
    "os/exec"
    "log"
    "bytes"
    "encoding/json"
    "strings"
)

type DockerExec interface {
    Provision()   (map[string] interface{},error)
    Bind()    (map[string] interface{},error)
    Unbind(serviceurl string)      (map[string] interface{},error)
    Deprovision() (map[string] interface{},error)
    
    Init(*DockerClient,string,*brokerapi.ImageDefinition) DockerExec
}

var CommandExecutors = map[string] DockerExec { 
    "DockerCommandExec" : DockerCommandExec{},
    }


type DockerCommandExec struct {
    client *DockerClient
    cId    string
    image *brokerapi.ImageDefinition
}

func (dcexec DockerCommandExec) Init(client *DockerClient ,cId string,image *brokerapi.ImageDefinition) DockerExec {
    return &DockerCommandExec{client,cId,image}
}

func (dcexec *DockerCommandExec) ExecIn (command []string) (map[string] interface{}, error) {
    execargs := strings.Split( dcexec.client.ServiceAgent.ExecArgs, "," )
    if execargs[0] == "" {
      execargs = []string{}
    }
    execargs = append( execargs, "docker-enter", dcexec.cId )
    execargs = append( execargs, command... )

    log.Println( "ExecIn.args[0]:", execargs[0] )
    log.Println( "ExecIn.args[:]:", execargs )

    cmd := exec.Command(execargs[0], execargs[1:]...)
    log.Println( "ExecIn.cmd:", cmd)        
    var out bytes.Buffer
    cmd.Stdout = &out
    var errout bytes.Buffer
    cmd.Stderr = &errout
    err := cmd.Run()

    log.Println( "ExecIn: cmd.Run() :", err," - ",out.String(), " - ", errout.String())
    if err != nil && err.Error() != "exit status 1" {
        log.Println("Error running cmd: ", err, " : ", out.String(), errout.String())
        return nil, err
    }

    var response map[string]interface{}
    resp := strings.TrimSpace( out.String() )

    log.Printf("ExecIn: return: '%q'\n", resp )
    if resp == "" {
        response = nil
    } else {
        err = json.Unmarshal([]byte(out.String()), &response)
        if err != nil {
            log.Println("ExecOnContainer: Unmarshall error: ", err)
            return response, err
        }
    }

    log.Println("ExecIn: Unmarshal response: ", response)
    return response, nil
}

func (dcexec DockerCommandExec) Provision()  (map[string] interface{}, error) {
    response,err := dcexec.ExecIn([]string{ "/provision" })
    return response,err
}

func (dcexec DockerCommandExec) Bind()  (map[string] interface{}, error) {
    response,err := dcexec.ExecIn([]string{ "/bind" })
    return response,err
}

func (dcexec DockerCommandExec) Unbind(serviceurl string)  (map[string] interface{}, error) {
    return dcexec.ExecIn([]string{ "/unbind" , serviceurl })
}

func (dcexec DockerCommandExec) Deprovision() (map[string] interface{}, error){
    return dcexec.ExecIn([]string{ "/deprovision" })
}

