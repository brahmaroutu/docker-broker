package brokerapi

import (
    "database/sql"
    "log"
    _ "github.com/go-sql-driver/mysql"
    _ "github.com/mattn/go-sqlite3"
    _ "github.com/lib/pq"
    "time"
    "strings"
    "strconv"
    "encoding/gob"
    "encoding/json"
    "bytes"
)

type Persister struct {
    Driver string
    Host string
    Port int
    User string
    Password string
    Database string
    Db  *sql.DB;
}

const (
    MYSQL = 1
    SQLITE = 2
    POSTGRES = 3
)

func (persister *Persister) getDBType() int {
    switch persister.Driver {
    case "mysql","MYSQL" :
        return MYSQL
    case "sqlite3","SQLITE3" :
        return SQLITE
    case "postgres","POSTGRES" :
        return POSTGRES     
    }
    return -1    
}

func (persister *Persister) UrlString4MySQL() (string) {
    return persister.User+":"+persister.Password+"@tcp("+persister.Host+":"+strconv.Itoa(persister.Port)+")/"+persister.Database
}

func (persister *Persister) UrlString4Postgres() (string) {
    return "postgres://"+persister.User+":"+persister.Password+"@"+persister.Host+":"+strconv.Itoa(persister.Port)+"/"+persister.Database+"?sslmode=disable"
}

func (persister *Persister) Connect() (error) {
    var err error   
    switch persister.getDBType() {
    case MYSQL :
        persister.Db, err = sql.Open(persister.Driver, persister.UrlString4MySQL())
    case SQLITE :
        persister.Db, err = sql.Open(persister.Driver, persister.Database)
    case POSTGRES :     
        persister.Db, err = sql.Open(persister.Driver, persister.UrlString4Postgres())
    }
    return err        
}

// generic functions

func (persister *Persister) InsertTable(tablename string, colmap map[string] interface{}) error {
    var insertCols string = "("
    var insertParams string = "("
    var insertVals []interface{}
    for colname := range colmap {
        if colmap[colname] == nil {
            log.Println("colname ",colname," has nil value")
            continue
        } 
        insertCols = insertCols + colname +  ","
        insertParams = insertParams + "?,"
        value := new(interface{})
        *value = colmap[colname]
        insertVals = append(insertVals,value)
    }
    insertCols = strings.TrimSuffix(insertCols,",")+")"
    insertParams = persister.parameterize(strings.TrimSuffix(insertParams,",")+")")
    stmt,err := persister.Db.Prepare("insert into "+tablename+insertCols+" values "+insertParams)
    if err != nil {
        return err
    }
    defer stmt.Close()

    _, err = stmt.Exec(insertVals...)
    return err
}

func (persister *Persister) UpdateTable(tablename string, colmap map[string] interface{}, cond string) error {
    var updateCols string = " set "
    var updateVals []interface{}
    for colname := range colmap {
        if colmap[colname] == nil {
            log.Println("colname ",colname," has nil value")
            continue
        } 
        updateCols = updateCols + colname + persister.parameterize("=?") + ","
        value := new(interface{})
        *value = colmap[colname]
        updateVals = append(updateVals,value)
    }
    updateCols = strings.TrimSuffix(updateCols,",")
    stmt,err := persister.Db.Prepare("update "+tablename+updateCols+" where "+cond)
    if err != nil {
        return err
    }
    defer stmt.Close()

    _, err = stmt.Exec(updateVals...)
    return err
}

func (persister *Persister) GetCount(tablename, cond string) (int32,error) {
    var count int32
    query := "SELECT count(*) FROM "+tablename
    if len(cond) > 0 {
        query = query+" where "+cond
    }
    err := persister.Db.QueryRow(query).Scan(&count)
    return count,err
}

func (persister *Persister) HasEntry(tablename, cond string) (bool) {
    count,err := persister.GetCount(tablename,cond)
    if err != nil {
        log.Println(err)
        return false
    }
    if (count > 0) {
        return true
    }
    return false    
}

//service agent calls

func (persister *Persister) GetServiceAgentList(cond string) ([]ServiceAgent,error) {
    var rows *sql.Rows
    var err error
    if (cond == "") {
        rows, err = persister.Db.Query("SELECT service_host,docker_host,docker_port,is_active,perf_factor,ping_interval_secs,last_ping,exec_command,exec_args,portbinding_min,portbinding_max,portbindings FROM serviceagents")
    } else {
        rows, err = persister.Db.Query("SELECT service_host,docker_host,docker_port,is_active,perf_factor,ping_interval_secs,last_ping,exec_command,exec_args,portbinding_min,portbinding_max,portbindings FROM serviceagents where "+cond)
    }
    if err != nil {
        return nil,err
    }
    defer rows.Close()
    
    var serviceagents []ServiceAgent  
    i:=0
    for rows.Next() {
        serviceagent := ServiceAgent{}
        var timevalue interface{}
        rows.Scan(&serviceagent.ServiceHost,&serviceagent.DockerHost,&serviceagent.DockerPort,&serviceagent.IsActive,&serviceagent.PerfFactor,&serviceagent.KeepAlive,&timevalue,&serviceagent.ExecCommand,&serviceagent.ExecArgs,&serviceagent.Portbind_min,&serviceagent.Portbind_max,&serviceagent.Portbindings)
        switch timevalue.(type) {
            case string:
                serviceagent.LastPing,err = time.Parse("2006-01-02 15:04:05 ",timevalue.(string))
            case time.Time:
                serviceagent.LastPing = timevalue.(time.Time)
            }
        
        if err != nil {
            log.Println("error reading row ",err)
        }
        serviceagents = append(serviceagents,serviceagent)
        i=i+1
    }
    return serviceagents,nil
}

func (persister *Persister) GetPortBindings(cond string) (int,int,[]int,error) {
    var pb_min,pb_max int
    var pb_bytes []byte
    var pb []int
    rows, err := persister.Db.Query("select portbinding_min,portbinding_max,portbindings from serviceagents where "+cond)
    if err != nil {
        return pb_min,pb_max,pb,err
    }
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(&pb_min,&pb_max,&pb_bytes)
        if len(pb_bytes) > 0 {
            pb = UnmarshalIntArray(pb_bytes)
        }
        return pb_min,pb_max,pb,err
    }
    
    return pb_min,pb_max,pb,sql.ErrNoRows
}

func (persister *Persister) WritePortBinding (ports []int, cond string) error {
    if persister.HasEntry("serviceagents",cond) {
        return persister.UpdateTable("serviceagents",map[string] interface{} {    
                                     "portbindings":MarshalIntArray(ports)},cond)
    } else {
        return sql.ErrNoRows
    }
}

func (persister *Persister) AddServiceAgents(serviceagents []ServiceAgent) error {
    var reterr error
    for _,sa := range serviceagents {
       err := persister.InsertTable("serviceagents",map[string] interface{} {"service_host":sa.ServiceHost,
                                                                           "docker_host":sa.DockerHost,
                                                                        "docker_port":sa.DockerPort,
                                                                        "last_ping":time.Now(),
                                                                        "is_active":sa.IsActive,
                                                                        "ping_interval_secs":sa.KeepAlive,  
                                                                        "exec_command":sa.ExecCommand,
                                                                        "exec_args":sa.ExecArgs,  
                                                                        "portbinding_min":sa.Portbind_min,    
                                                                        "portbinding_max":sa.Portbind_max,    
                                                                        "perf_factor":sa.PerfFactor})
        if err != nil {
            reterr = err
        }
    }
    return reterr
}

func (persister *Persister) MarkServiceAgentInactive(Host string) error {
    return persister.UpdateTable("serviceagents",map[string] interface{} {"is_active":false},"docker_host='"+Host+"'")
}

func (persister *Persister) MarkServiceAgentActive(Host string) error {
    return persister.UpdateTable("serviceagents",map[string] interface{} {"is_active":true},"docker_host='"+Host+"'")
}

func (persister *Persister) AddorUpdateServiceAgent(sa ServiceAgent) error {
    if persister.HasEntry("serviceagents","docker_host='"+sa.DockerHost+"'") {
        persister.UpdateTable("serviceagents",map[string] interface{} {"last_ping":time.Now(),
                                                                       "service_host":sa.ServiceHost,
                                                                       "docker_port":sa.DockerPort,
                                                                       "is_active":sa.IsActive,
                                                                       "ping_interval_secs":sa.KeepAlive,  
                                                                       "exec_command":sa.ExecCommand,
                                                                       "exec_args":sa.ExecArgs,  
                                                                       "portbinding_min":sa.Portbind_min,    
                                                                       "portbinding_max":sa.Portbind_max,    
                                                                       "perf_factor":sa.PerfFactor},"docker_host='"+sa.DockerHost+"'")
        return nil
    } else {
        return persister.AddServiceAgents([]ServiceAgent{sa})
    }
}

//service instance calls

func (persister *Persister) GetServiceAgentFromInstance(cond string) (string,error) {
    var serviceagent string
    rows, err := persister.Db.Query("select service_agent from serviceinstances s where "+cond)
    if err != nil {
        return serviceagent,err
    }
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(&serviceagent)
        return serviceagent,err
    }
    
    return serviceagent,sql.ErrNoRows
}

func (persister *Persister) GetServiceUrl(cond string) string {
    var serviceurl string
    rows, err := persister.Db.Query("select service_url from serviceinstances  where "+cond)
    if err != nil {
        return serviceurl
    }
    defer rows.Close()
    
    for rows.Next() {
        rows.Scan(&serviceurl)
        if err != nil {
            log.Println("error reading row ",err)
        }
        return serviceurl
    }
    return serviceurl
}

func (persister *Persister) AddServiceInstance(service_name string, service_port, host_port int, 
            service_url, container_id, service_agent,container_name,image_name string, 
            pr ProvisioningRequest, started_at time.Time) error {
       return persister.InsertTable("serviceinstances",map[string] interface{} {"service_name":service_name,
                                                                        "service_port":service_port,
                                                                       "mapped_host_port":host_port,
                                                                        "service_url":service_url,
                                                                        "container_id":container_id,  
                                                                        "service_agent":service_agent,
                                                                        "container_name":container_name,  
                                                                        "image_name":image_name,    
                                                                        "cf_instance_id":pr.InstanceId,    
                                                                        "cf_plan_id":pr.PlanId,    
                                                                        "cf_org_id":pr.OrgId,    
                                                                        "cf_space_id":pr.SpaceId,    
                                                                        "started_at":started_at})
    

}

func (persister *Persister) DeleteServiceInstance(instanceId string) error {
    stmt, err := persister.Db.Prepare("delete from serviceinstances where cf_instance_id"+persister.parameterize("=?"))
    if err != nil {
        return err
    }
    defer stmt.Close()
    _, err = stmt.Exec(&instanceId)
    return err     
}

func (persister *Persister) ReadContainers(container_map map[string]string) error {
    rows, err := persister.Db.Query("select container_id,cf_instance_id from serviceinstances")
    if err != nil {
        return err
    }
    defer rows.Close()
    
    var containerid,instanceid string  
    
    for rows.Next() {
        rows.Scan(&containerid,&instanceid)
        if err != nil {
            log.Println("error reading row ",err)
            continue
        }
        //we should check before we update, later
        container_map[instanceid] = containerid
    }
    return nil
}

func (persister *Persister) GetContainerIdAndImageName(instanceid string) (string,string) {
    var containerid,imagename string  
    err := persister.Db.QueryRow("select container_id,image_name from serviceinstances where cf_instance_id"+persister.parameterize("=?"),instanceid).Scan(&containerid,&imagename)
    if err != nil {
        log.Println("Failed to get values",err) 
        return "",""
    }
    
    return containerid,imagename
}

func (persister *Persister) GetServicePort(instanceid string) int {
    var service_port int 
    err := persister.Db.QueryRow("select service_port from serviceinstances where cf_instance_id"+persister.parameterize("=?"),instanceid).Scan(&service_port)
    if err != nil {
        log.Println("Failed to get values",err) 
        return -1
    }
    
    return service_port
}

//service bindings calls

func (persister *Persister) AddServiceBinding(instanceId,bindingId,appId string, started_at time.Time) error {
       return persister.InsertTable("servicebindings",map[string] interface{} {"cf_instance_id":instanceId,    
                                                                        "cf_binding_id":bindingId,    
                                                                        "cf_app_id":appId,    
                                                                        "started_at":started_at})
} 

func (persister *Persister) DeleteServiceBinding(instanceId,bindingId string) error {
    stmt, err := persister.Db.Prepare("delete from servicebindings where "+persister.parameterize("cf_instance_id=? and cf_binding_id=?"))
    if err != nil {
        return err
    }
    defer stmt.Close()
    result, err := stmt.Exec(&instanceId,&bindingId)
    if err != nil {
    }
    if i,_ := result.RowsAffected(); i == 0 {
        return sql.ErrNoRows
    }
    return err
     
}


//service configurations calls

func (persister *Persister) AddServiceConf(user,password,catalog string) error {
      return persister.InsertTable("serviceconfigurations",map[string] interface{} {"username":user,    
                                                                        "password":password,    
                                                                        "catalog":catalog})
} 

func (persister *Persister) GetServiceId(catalog string) int {
    var id int  
    err := persister.Db.QueryRow("select id from serviceconfigurations where catalog"+persister.parameterize("=?"),catalog).Scan(&id)
    if err != nil {
        log.Println("Failed to get values",err) 
        return -1
    }
    
    return id    
}

func (persister *Persister) GetServiceConf() ([]ServiceDefinition,error) {
    var rows *sql.Rows
    var err error
    rows, err = persister.Db.Query("select id,username,password,catalog, name,plan,numinstances,containername, dashboardurl,credentials from serviceconfigurations sc,imageconfigurations ic where ic.service_id=sc.id order by sc.id")
    if err != nil {
        return nil,err
    }
    defer rows.Close()
    
    svcdefs := make(map[int] *ServiceDefinition)
    for rows.Next() {
        var rowid int
        svcdef := ServiceDefinition{}
        imgdef := ImageDefinition{}
        //interestingly scan does not work if we scan like numinstances after the maps(deprovision), I had to move those field to the top
        var dburl,creds string
        err = rows.Scan(&rowid,&svcdef.User,&svcdef.Password,&svcdef.Catalog, &imgdef.Name,&imgdef.Plan,&imgdef.Numinstances,&imgdef.Containername, &dburl,&creds)
        if err != nil {
            log.Println("error reading row ",err)
        }
        if len(dburl) > 0 {
            json.Unmarshal([]byte(dburl),&imgdef.DashBoardUrl)        
        }
        if len(creds) > 0 {
            json.Unmarshal([]byte(creds),&imgdef.Credentials)        
        }
        if currsvcdef,ok := svcdefs[rowid]; ok {
            currsvcdef.Images = append(currsvcdef.Images,imgdef)
        } else {
            svcdefs[rowid] = &svcdef
            svcdef.Images = append(svcdef.Images,imgdef)
        }
            
    }
    
    ret_svcdefs := make([]ServiceDefinition,0,len(svcdefs))
    for svc := range svcdefs {
        ret_svcdefs = append(ret_svcdefs,*svcdefs[svc])
    }
    return ret_svcdefs,nil
}

func (persister *Persister) DeleteServiceConf(id int) error {
    stmt, err := persister.Db.Prepare("delete from serviceconfigurations where "+persister.parameterize("id=?"))
    if err != nil {
        return err
    }
    defer stmt.Close()
    result, err := stmt.Exec(&id)
    if err != nil {
    }
    if i,_ := result.RowsAffected(); i == 0 {
        return sql.ErrNoRows
    }
    return err
}

//image configurations calls

func (persister *Persister) AddBasicImageConf(service_id int, name,plan string) error {
    numinstances := 1
    return persister.InsertTable("imageconfigurations",map[string] interface{} {"service_id":service_id,    
                                                                        "name":name,    
                                                                        "plan":plan,    
                                                                        "numinstances":numinstances})

} 

func (persister *Persister) AddImageConf(service_id int, name,plan,dashboardurl, credentials string, numinstances int,containername string) error {
    return persister.InsertTable("imageconfigurations",map[string] interface{} {"service_id":service_id,    
                                                                        "name":name,    
                                                                        "plan":plan,    
                                                                        "dashboardurl":dashboardurl,    
                                                                        "credentials":credentials,    
                                                                        "containername":containername,    
                                                                        "numinstances":numinstances})
} 

func (persister *Persister) DeleteImageConf(service_id int,name,plan string) error {
    stmt, err := persister.Db.Prepare("delete from imageconfigurations where "+persister.parameterize("service_id=? and name=? and plan=?"))
    if err != nil {
        return err
    }
    defer stmt.Close()
    result, err := stmt.Exec(&service_id,&name,&plan)
    if err != nil {
    }
    if i,_ := result.RowsAffected(); i == 0 {
        return sql.ErrNoRows
    }
    return err
}

//broker certs calls

func (persister *Persister) AddBrokerCertsConf(agent string,clientcertfile,clientkeyfile,cafile []byte) error {
    return persister.InsertTable("brokercertificates",map[string] interface{} {"serviceagent":agent,    
                                                                        "clientcertfile":clientcertfile,    
                                                                        "clientkeyfile":clientkeyfile,    
                                                                        "cafile":cafile})
} 

func (persister *Persister) GetBrokerCerts() ([]BrokerCerts, error) {
   var rows *sql.Rows
    var err error
    var certs []BrokerCerts
    rows, err = persister.Db.Query("select serviceagent,clientcertfile,clientkeyfile,cafile from brokercertificates")
    if err != nil {
        return certs,err
    }
    defer rows.Close()
    
    for rows.Next() {
        brokercerts := BrokerCerts{}
        rows.Scan(&brokercerts.Host,&brokercerts.ClientCert,&brokercerts.ClientKey,&brokercerts.CA)        
        if err != nil {
            log.Println("error reading row ",err)
        }
        certs = append(certs,brokercerts)
        
    }
    return certs,err

}

func (persister *Persister) DeleteBrokerCertsConf(agent string) error {
    stmt, err := persister.Db.Prepare("delete from brokercertificates where "+persister.parameterize("serviceagent=?"))
    if err != nil {
        return err
    }
    defer stmt.Close()
    result, err := stmt.Exec(&agent)
    if err != nil {
    }
    if i,_ := result.RowsAffected(); i == 0 {
        return sql.ErrNoRows
    }
    return err
}



func (persister *Persister) TimeElapsed(eventtime string) string {
    switch persister.getDBType() {
    case MYSQL :
        return "time_to_sec(timediff(utc_timestamp(),"+eventtime+"))"
    case SQLITE :
        return "(julianday() - julianday("+eventtime+"))*24*3600"
    case POSTGRES :     
        return "extract (epoch from (current_timestamp::timestamp - "+eventtime+"::timestamp))"
    }
    return "9999"
}

func (persister *Persister) parameterize(sqlstr string) string {
    if persister.getDBType() != POSTGRES {
        return sqlstr
    } 
    strs := strings.Split(sqlstr,"?")
    var retval string
    var count int=0
    for _,str := range strs {
        if count > 0 && count < len(strs) {
            retval = retval+"$"+strconv.Itoa(count)+str
        } else {
        retval = retval + str
        }
        count++
    }
    return retval
}

func MarshalIntArray(inta []int) []byte {
    buffer := new(bytes.Buffer)
    e := gob.NewEncoder(buffer)
    err := e.Encode(inta)
    if err != nil {
        log.Println("Failed to marshal ", err)
    }
    return buffer.Bytes()    
}

func UnmarshalIntArray(ba []byte) []int {
    buffer := bytes.NewBuffer(ba)
    var inta = new([]int)
    d := gob.NewDecoder(buffer)
    err := d.Decode(&inta)
    if err != nil {
        log.Println("Failed to unmarshal ", ba, err)
    }
    return *inta
}
