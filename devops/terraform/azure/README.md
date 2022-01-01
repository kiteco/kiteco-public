# Azure TF Readme

setup and general info for azure devops stuff

## General TF Setup

1. The following variables should be defined in a file you can source into your terraform execution environment like so:
```
#!/bin/sh
export TF_VAR_az_subscription_id="..."
export TF_VAR_az_client_id="..."
export TF_VAR_az_client_secret="..."
export TF_VAR_az_tenant_id="..."
export TF_VAR_localfiles_db_password="..."
az account set --subscription $TF_VAR_az_subscription_id
```

## Dev Region Initial Setup

1. run `all.sh` to apply tf defs to the dev region(s)


## Prod Region Initial Setup

1. run `all.sh` to apply tf defs to the various prod region(s)
2. manually create application load balancers and make sure you specify the path to the correct tls certificates
```
alb_create.sh "<region>" "prod" <path-to-wildcard-cert>
alb_create.sh "<region>" "staging" <path-to-wildcard-cert>
```
3. grab IPs and configure your `~/ssh/config` file to contain something like the following for each region you'd like to connect to:
```
Host bastion-eastus
  HostName <bastion-public-ip-address>
  User ubuntu
  IdentityFile <path-to-your-dev-ssh-priv-key>

Host vpntunnel-eastus
  HostName <any-internal-address>
  User ubuntu
  IdentityFile <path-to-your-dev-ssh-priv-key>
  ProxyCommand ssh ubuntu@bastion-eastus -W %h:%p
```
4. configure (if necessary) and use fabric to provision vpn tunnel machines. For each region:
    1. edit fabfile.py to use ssh config by adding `env.use_ssh_config = True` if using `~/ssh/config` to specify hosts
    2. run `fab vpntunnel-eastus strongswan.initialize`
    3. run `fab vpntunnel-eastus strongswan.put_cfg:../azure/ipsec.useast.conf` but be sure to use the correct config for your region
    4. copy the pre shared key for the tunnel into `/etc/ipsec.secrets` on the box, it should look something like `%any %any 'some_key_here'` or it can be more explicit like `%useast %dev 'some_key'`
    5. run `fab vpntunnel-eastus strongswan.restart`
    6. run `fab vpntunnel-eastus strongswan.status` and you should see information about the ipsec tunnel status, if you do not get any response the ipsec daemon is not running correctly. A working tunnel will display both subnets and something like "TUNNEL ESTABLISHED"
    7. **optional** if you dont get a response from status then you will need to log into the box manually and restart the process with `sudo ipsec restart`, I don't know why fabric remote restart fails sometimes



