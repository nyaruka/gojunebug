angular.module('junebug', [])
    .controller('EndpointController', function($scope, $http) {
        var ec = this;


        $scope.formData = {}
        $scope.endpoints = [];

        $scope.formData.connUuid = "connection uuid";
        $scope.formData.msgId = "msg id";

        $scope.endpoints.push({
            id: 'list-conns',
            expanded: false,
            method: 'GET',
            url: '/connection',
            url_parts: [{ value: '/connection' }],
            title: "List active connections",
            has_body: false,
            response: ""
        });

        $scope.endpoints.push({
            id: 'read-conn',
            expanded: false,
            method: 'GET',
            url: '/connection/[conn uuid]',
            url_parts: [
                { value: '/connection/'},
                { name: 'conn_uuid', length: 36 }],
            title: "Get status for specific connection",
            has_body: false,
            response: ""
        });

        $scope.endpoints.push({
            id: 'add-conn',
            expanded: false,
            method: 'PUT',
            url: '/connection',
            url_parts: [{ value: '/connection' }],
            body: angular.toJson(
                {
                    senders: {
                        "type": "echo",
                        "count": 5,
                        "config": {
                            "pause": "1"
                        }
                    },
                    "receivers": {
                        "receiver_type": "http",
                        "count": 5,
                        "config": {
                            "url": "http://myhost.com/receive"
                        }
                    }
                } , true),
            body_rows: 17,
            title: "Create new connection",
            has_body: true,
            response: ""
        });

        $scope.endpoints.push({
            id: 'delete-conn',
            expanded: false,
            method: 'DELETE',
            url: '/connection/[conn uuid]',
            url_parts: [
                { value: '/connection/'},
                { name: 'conn_uuid', length: 36}],
            title: "Deletes a connection",
            has_body: false,
            response: ""
        });

        $scope.endpoints.push({
            id: 'add-msg',
            expanded: false,
            method: 'PUT',
            url: '/connection/[conn uuid]/send',
            url_parts: [
                { value: '/connection/'},
                { name: 'conn_uuid', length: 36},
                { value: '/send'}],
            title: "Sends a message",
            has_body: true,
            body: angular.toJson(
                {
                    address: "+250788123123",
                    text: "Hello World"
                } , true),
            body_rows: 5,
            response: ""
        });

        $scope.endpoints.push({
            id: 'read-msg',
            expanded: false,
            method: 'GET',
            url: '/connection/[conn uuid]/status/[msg id]',
            url_parts: [
                { value: '/connection/'},
                { name: 'conn_uuid', length: 36},
                { value: '/status/' },
                { name:'msg_id', length: 18 }],
            title: "Gets the status of a message",
            has_body: false,
            response: ""
        });

        $scope.connection = null;

        ec.toggle = function(ep){
            ep.expanded = !ep.expanded
        };

        function findField(data, name){
            if (typeof data != "object"){
                return null;
            }

            for(var key in data) {
                if (data.hasOwnProperty(key)) {
                    if (key == name){
                        return data[key];
                    } else {
                        var found = findField(data[key], name);
                        if (found != null) {
                            return found
                        }
                    }
                }
            }
            return null;
        }

        ec.fire = function(ep){
            ep.response = "";

            // build our URL from our parts
            var url = "";
            for(var i in ep.url_parts){
                var part = ep.url_parts[i];
                if (part.name == 'conn_uuid'){
                    url += $scope.formData.connUuid;
                } else if (part.name == 'msg_id'){
                    url += $scope.formData.msgId;
                } else {
                    url += part.value
                }
            }

            var req = { method: ep.method, url: url, headers: {'Content-Type': 'application/json' }} ;
            if (ep.has_body){
                req['data'] = ep.body;
            }
            $http(req).
                success(function(data, status) {
                    // walk the tree to see if there is a uuid to set
                    var foundConn = findField(data, "uuid");
                    if (foundConn != null) {
                        $scope.formData.connUuid = foundConn;
                    }

                    var foundMsg = findField(data, "id");
                    if (foundMsg != null){
                        $scope.formData.msgId = foundMsg;
                    }

                    ep.response = "STATUS: " + status + "\n" + angular.toJson(data, true);

                    // highlight it
                    var response = $("#" + ep.id + "-response");
                    $(response).text(ep.response);
                    hljs.highlightBlock(response.get(0));
                }).
                error(function(data, status) {
                    ep.response = "ERROR: " + status + "\n" + data;

                    var response = $("#" + ep.id + "-response");
                    $(response).text(ep.response);
                    hljs.highlightBlock(response.get(0));
                });
        };
  });
