var http = require('http');
var mysql = require('mysql');
var express = require('express');
var app = express();
var exec = require('child_process').execFile;

var mysqlurl
if (process.env.VCAP_SERVICES) {
    var vcapenv = JSON.parse(process.env.VCAP_SERVICES);
    console.log("env")
    var found=false
    var vcapkeys = Object.keys(vcapenv)
    vcapkeys.forEach(function(vcapkey) {
        newObj = vcapenv[vcapkey]
        //crude, ok for now as it has to work with various service names
        tempjson = vcapenv[vcapkey][0];
        mysqlurl = tempjson["credentials"]
        return
    })
    console.log("mysqlurl")
    console.log(mysqlurl)
} else {
    var connStr = '{ \
            "name": "mytest",\
            "hostname": "localhost",\
            "host": "localhost",\
            "port": 3306,\
            "user": "docker",\
            "username": "docker",\
            "password": "docker",\
            "uri": "mysql://docker:docker@localhost:3306/mytest",\
            "database": "mytest"\
         }';
     mysqlurl = JSON.parse(connStr);
}
app.set('mysql_url', mysqlurl);
app.set('port', process.env.PORT || 3000);


console.log(app.get('mysql_url'));
// Create a connection to MySql Server and Database
var connection = mysql.createConnection(
    app.get('mysql_url')
);

app.get('/hello', function(req, res){
  res.send('Hello World'+JSON.stringify(app.get('mysql_url')));
});

app.get('/petinfo', function(req, res) {
    connection.query("SELECT * from PET", function(err, rows) {
        if (err != null) {
            res.end("Query error:" + err);
        } else {
            res.send(JSON.stringify(rows));
        }
    });
});

var populate_tables=function () {
    fs = require('fs')
    fs.readFile('./setup.sql', 'utf8', function (err,data) {
        if (err) {
            return console.log(err);
        }
        var stmts = data.split(";")
        for (i=0;i<stmts.length;i++) {
            connection.query(stmts[i], function(err, rows) {
                if (err != null) {
                    console.log("Query error:" + err);
                } else {
                    console.log("Success in setup")
                }
            });
         }
    });
}

var server = app.listen(app.get('port'), function() {
//http.createServer(app).listen(app.get('port'), function(){
    connection.connect(function(err) {
        if(err != null) {
            console.log('Error connecting to mysql:' + err+'\n');
        }
    });
    populate_tables();
//   console.log('Express server listening on port ' + app.get('port'));
    console.log('Listening on port %d', server.address().port);
});
