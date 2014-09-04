CloudFoundry Docker Broker
================
CloudFoundry Docker Broker implements the Cloud Foundry Service broker API to provide management of Docker-based service instances that can be bound to an application running in Cloud Foundry.  Some of the main features of CloudFoundry Docker Broker include:
* Can be run anywhere; as a standalone executable, as a CloudFoundry app or inside a Container
* Supports static as well as service generated credentials
* Supports static, multi-tenant as well as pre-existing service instances
* TLS/SSL support 
* Can be scaled to support large number of CloudFoundry applications as well as any number of Docker hosts

The basic architecture is as follows:
![Broker Architecture](docs/arch.png)

There are two main components: a Broker and Agents:
Broker
------
The Broker is the component that interacts with CloudFoundry and manages the lifecycle of the CloudFoundry service instances. More than one instance of a Broker can be executed at a time, thus allowing it to scale as needed. All of its persistence is stored in a DB that is shared across all Broker instances.

As CloudFounder service instance operations are requested, the Broker will interact directly with a Docker host. For example, during the provisioning request of a new service instance, the Broker will choose ones the available Docker hosts to create a new Docker container for the service instance.

The Broker's configuration contains the list of available service instances, represented by Docker images. See the details about the Broker's configuration file for more details.

Agents
------
Agents act as the intermediaries between the Broker and Docker. Each Docker host must have exactly one Agent associated with it, normally running on the same host as Docker itself. The Agent's job is to periodically ping the Broker to let it know that this Docker host is available for use. As new Docker hosts are created, its associated Agent is responsibe for registering with a Broker.

Aside from registering a Docker host with the Broker, the Agent will also tell the Broker how to communicate with the Docker on that host since each Docker host may require a slightly different communication mechanism.  See the Agent's configuration file for more details.


Quick Start Guide
-----------------
To get the CloudFoundry Docker Broker running and connected to your Cloud Foundry environment, do the following:
* Get a fresh Ubuntu VM and sudo to root
* `curl -k http://github.com/brahmaroutu/docker-broker/raw/master/bin/install | bash`
This will install Docker, download the CloudFoundry Service Broker and compile it. If 'curl' is not available then do the following first: `apt-get update && apt-get install -y curl`

Note: the above works for Ubuntu. For other operating system you may need to modify the `install` script first.

By default the configuration files are setup to use the IP address on the eth0 adapter as the IP address to use for your CloudFoundry service instances. You can modify this by editing the `servicehost` property in the config/agent.config file.

To start the database, Broker and Agent, run:
```
cd src/github.com/brahmaroutu/docker-broker
bin/start
```
This brings all 3 up on this one VM. The Broker is listening on port 9998 by default. You are now ready to connect it to a CloudFoundry instance. Note: its critical that your CloudFoundry install can reach the Broker and the Docker containers, so make sure the IP address used for the `servicehost` in the agent.config and the IP address used in the create-service-broker command below is accessible to your CloudFoundry install and apps.

First create a new service broker, for the purposes of this example we will use `192.168.1.1` as the IP of the Ubuntu VM:
```
cf create-service-broker dockerBroker admin admin http://192.168.1.1:9998
```

Now make the available services visible:
```
cf enable-service-access mysql-docker -p 100 -o <\your_org_name\>
cf enable-service-access mysqlstatic-docker -p 100 -o \your_org_name\>
```

You should now be able to see and use these new services, try `cf m` to make sure they're visble to you.

The two services available are both `mysql` but the `mysqlstatic` one uses a set of predefined credentials, while the other one will ask the new service instance (Docker container) to create a new random set of credentials for you.

Adding New Services
===================
Brokers provide the capability to invoke CloudFoundry Lifecycle commands that gets called during the CloudFoundry provisioning, bind, unbind and deprovision steps. During each of the CloudFoundry provision steps, the Broker will invoke the corresponding `/provision`, `/bind`, `/unbind` and `/deprovision` executable found in the Docker container for the service instance. Each script is responsible for taking any actions necessary to prepare the service for that stage of the lifecycle - for example: 

* `/provision` may create a new database instance, while `/deprovision` may delete it.
* `/bind` may create credentials to access the service and returns those credentials as a JSON object to stdout.

Service definitions can be added at runtime using REST API. Initial services are preloaded from the config file and stored in the persister. Once the database is loaded only way to add new services is through the REST API. See details on the REST API under Broker API section.

Broker API
==========
The follow describes the Broker's REST APIs:

Endpoint | Description
-------- | -----------
/v2/catalog | Returns the catalog of service available from the Broker - supports GET.
/v2/service_instances/\<serviceID\> | Manage service instances - supports PUT to provision and DELETE to deprovision.
/v2/service_instances/\<serviceID\>/service_bindings/\<bindingID\> |  Manage the service instance bindings to applications - supports PUT to bind and DELETE to unbind.
/ping | Agents will POST to this endpoint to register, and periodically 'ping' the Broker.  Ping format should be {"ServiceHost":"localhost", "DockerHost":"localhost", "DockerPort":1234, "LastPing":"2014-09-02T20:38:21.734559269-07:00", "IsActive":true, "PerfFactor":1, "KeepAlive":1, "ExecCommand":"", "ExecArgs":"", "Portbind_min":0, "Portbind_max":0, "Portbindings":null} See the Agent config file section for details about these fields.
/\<catalogName\>/image/\<imageName\> | Endpoint used to manage services available from a Broker - supports GET, PUT and DELETE. PUT (create) request body for adding  a new service image should look like {"Name":"mysql", "Plan":"100", "Numinstances":1} See the Broker config file details for more information on these fields.  Note: if the `DashBoardURL` or `Credentials` properities are present then the Broker will assume the service has a set of static credentials and will not try to invoke the lifecycle executables in the container. If the `ContainerName` property is present then the `numInstances` property must be set to zero and the container identified by the `ContainerName` will be used rather than creating a new container for the service instances. E.g.: "DashBoardUrl":{"dashboard_url":"mysql://admin:admin@127.0.0.1:3306/myDB"},<br> "Credentials":{"database":"mysqlDB", "uri":"mysql://admin:admin@127.0.0.1:3306/myDB", "host":"127.0.0.1", "port":"1234", "username":"admin", "password":"admin"},<br> "Containername":"MyExistingContainerID"
/\<catalogName\>/images | List all service images in a catalog - supports GET.
/certificate/\<dockerHost\> | Add, list or delete certificates. For adding certs JSON should look like: {"Host":"\<dockerHost\>", "ClientCert":"\<clientCert\>", "ClientKey":"\<clientKey\>", "CA":"\<CA\>"}  Supports GET, PUT and DELETE.
/certificates | List all hosts that are registered with SSL certificates - supports GET.


Configuration Files
===================

broker.config
-------------
The following table describes the properties in the `broker.config` file:

Property | Desription
-------- | ----------
dispatcher | Use "SimpleDispatcher" for now. It randomly picks one of the available docker hosts.
listenIP | Binding IP to use for this Broker. Use `0.0.0.0` to allow all interfaces.
port | Listen port to use for this Broker.
 |
**persister** | Database used to store the Broker's configuration.
.driver | Type of DB - e.g. `mysql`
.host | Host IP (or name) to use to connect to the DB.
.port | Port to use to connect to the DB.
.user | User name to use when connecting to the DB.
.password | Password to use when connecting to the DB.
.database | DB name.
 |
**services** | List of services (Docker images) available. <br>Note, this section is only used when the DB is empty. Once the DB is populated you need to modify the list of available services via the Broker's REST API.
.user | User name to use for incoming REST requests to modify the list of services or agent authentication.  
.password | Password to use for incoming REST requests to modify the list of services.
.catalog | Name of the catalog
.images | List of Docker images to expose as CloudFoundry services.
.images.name | Name of the service to expose in CF.
.images.plan | CloudFoundry's "plan" value.
.images.dashboardurl | URL to the dashboard for this service.
.images.credentials | An optional set of credentials that the Broker should use for service instances. When set the Broker will NOT attempt to use the bind and provision lifecycle scripts within the Docker container.
.images.credentials.* | Service defined credentials.
.images.numinstance | Use "1" for now.

agent.config
------------
The following table describes the properties in the `agent.config` file:

Property | Description
-------- | -----------
**serviceagent** | Fields related to interacting with the Broker and how CloudFoundry apps should interact with the service instances
.servicehost | Host IP (or name) that the Broker should use within the credentials for services hosted on this Docker host. $HOST in the credentials will be substitued with this value.  Additionally, $PORT will be replaced with the Docker host port that is mapped to the services EXPOSE'd port.
.dockerhost | Host IP (or name) that the Broker should use when connecting to this Docker host.
.dockerport | Port that the Broker should use when connecting to this Docker host.
.isactive | Indicated whether this Docker host is available for new service instances.
.keepalive | The delay between each "ping" that the Agent sends to the Broker to indicate that it is still alive. This will also determine the amount of time the Broker waits before it considers the Agent/Docker-host to be dead - it is 3 times this value.
.ExecCommand | Use "DockerCommandExec" for now.
.ExecArgs | A comma separated list command line arguments that will be used to run the docker-enter command on this Docker host. These arguments make up the command that the Broker will use to talk to this Docker, so it may need to include an ssh command or sudo. The Broker will append "docker-enter" to the end of the list of arguments. The exact values will be based on your setup. For example, `sshpass,-tpassword,root@mydocker` or `boot2docker,ssh,sudo,`
.perffactor | For future use, Agent perffactor will let broker to chose agents with more available resources. Use "1.22" for now.
.portbind_min | The lowest port number that the Broker should use when exposing ports from containers through the Docker host.
.portbind_max | The highest port number that the Broker should use when exposing ports from containers through the Docker host.
 |
**brokerservers** | Fields related the Brokers that this Agent should connect to.
.host | Host IP  (or name) of the Broker to connect to.
.port | Port of the Broker to connect to.
.user | User name to use to connect to the Broker.
.password | Password to use to connect to the Broker.

Non-Quick Start Guide
=====================
* Compile the Broker and Agent - see "bin/install" - they are just GO apps:
  * to run broker: `./broker [ -config <filename> ]`
  * to run agent : `./agent  [ -config <filename> ] [ -clientcert <clientcertificate file name> -clientkey <clientkey file name> -cacert <rootcertificate file name> ]`
* Bring up as many Brokers as you want. Each is just an executable, and connect them all to the same persistence/DB
* Bring up as many Docker hosts a you want (ex. via BOSH). All each ones needs is Docker and and Agent. The Agent will connect to the Broker to make it aware of the new Docker host.  Critial piece is getting the correct ExecArgs so the Broker can talk to the Docker for nsenter.

Test Cases
==========
* All test cases are written using Ginkgo and Gomega. Test cases use sqlite3 as persister. To run the testcases first create the sqlite3 database using the command `sqlite3 broker_testdb < ../setup_sqlite.sql`. Then simply run `ginko` to run the tests.
* To run all testcases simply call `broker/run_tests`.

Outstanding Work Items:
=======================
* No log draining implemented. 
* Running multiple services instances in a single container is not supported
* Require many more testcases for better code converage
* Allow for tags to be specified on each image/service
