# kubengr
A tool for simplifying the installation of Kubernetes


## use kubengr command

Master:
```shell
./kubengr init --host-name <node-name>
```

Worker:
```shell
./kubengr join --host-name <node-name> --master-address <master-address> --token <token> --discovery_token_ca_cert_hash <discovery_token_ca_cert_hash>
```


## use kubengr.sh script

Master:
```shell
.kubengr.sh master <node-name> 
```

Worker:
```shell
.kubengr.sh worker <node-name>  <master-address> <token> <discovery_token_ca_cert_hash>
```