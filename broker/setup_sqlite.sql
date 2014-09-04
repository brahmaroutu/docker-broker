CREATE TABLE serviceagents (
        docker_host        VARCHAR(32) NOT NULL, 
        docker_port        INT,
        service_host       VARCHAR(32) NOT NULL, 
        last_ping          TIMESTAMP, 
        ping_interval_secs INT,
        is_active          BOOLEAN DEFAULT false, 
        perf_factor        DECIMAL(5,2),
        exec_command       VARCHAR(32),
        exec_args          VARCHAR(64),
        portbinding_min    INT default 49000,
        portbinding_max    INT default 49900,
        portbindings       BLOB,
        primary key        (docker_host));

CREATE TABLE serviceinstances (
        service_name       VARCHAR(32) NOT NULL, 
        service_port       INT, 
        mapped_host_port   INT, 
        service_url        VARCHAR(64), 
        container_id       VARCHAR(128),
        container_name     VARCHAR(128),
        image_name         VARCHAR(128), 
        cf_instance_id     VARCHAR(36) NOT NULL, 
        cf_plan_id         VARCHAR(36), 
        cf_org_id          VARCHAR(36),  
        cf_space_id        VARCHAR(36), 
        started_at         TIMESTAMP,
        service_agent      VARCHAR(15),
        primary key        (cf_instance_id));

CREATE TABLE servicebindings (
        cf_instance_id     VARCHAR(36) NOT NULL, 
        cf_app_id          VARCHAR(36) NOT NULL, 
        cf_binding_id      VARCHAR(36) NOT NULL, 
        started_at         TIMESTAMP,
        primary key        (cf_instance_id,cf_binding_id));

CREATE TABLE serviceconfigurations (
        id                 INTEGER PRIMARY KEY AUTOINCREMENT,         
        username           VARCHAR(36) NOT NULL,
        password           VARCHAR(36) NOT NULL,
        catalog            VARCHAR(36) NOT NULL);

CREATE TABLE imageconfigurations (
        service_id         INT NOT NULL,
        name               VARCHAR(36) NOT NULL,
        plan               VARCHAR(36) NOT NULL,
        dashboardurl       VARCHAR(255),
        credentials        VARCHAR(255),
        numinstances       INT,
        containername      VARCHAR(36),
        primary key        (service_id,name,plan),
        foreign key (service_id) references serviceconfigurations(id));
           
CREATE TABLE brokercertificates (
        serviceagent       VARCHAR(32),
        cafile             BLOB,
        clientcertfile     BLOB,
        clientkeyfile      BLOB);
