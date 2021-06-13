# Go Load Balancer

The purpose of this application is to create a load balancer which routes requests to two business servers. The logic is simple: Pass in some string and it will return the reversed string. If no server responds within 30 seconds, a failure response is given. The request will also need to be properly formatted. So feel free to mess around with the request structure.

## Make requests to the server from any computer

curl --header "Content-Type: application/json" --request POST --data '{ "data": "some string" }' http://ec2-35-183-71-175.ca-central-1.compute.amazonaws.com:8000

{"data":"gnirts emos"}

> Note: Port 8000 is the load balancer - this will send requests to the fastest responding server. Try sending a request to 8001 or 8002 to see the behaviour.

## How to run unit tests

1. Clone this repository
2. Request access for a .pem key. Email kyle.jensen72@gmail.com for access.
3. Store your .pem key in the default location (~/.ssh/pem/ec2-user.pem)
4. Run npm test to run the JEST testing suite

## How to run scripts

1. Make sure the bash scripts are executable. You can run the following command in the root folder:
```sh
$ chmod +x deploy_servers
$ chmod +x start_server
$ chmod +x stop_server
$ chmod +x check_logs
```
2. Run the appropriate command. You can get information about how to use each command by specifying the **--help** or **-H** flags
```sh
Examples:
$ ./deploy_servers
$ ./start_server --all
$ ./stop_server --help
```
> Note: These commands only work on the live production code at the moment - support for localhost may be added in the future
