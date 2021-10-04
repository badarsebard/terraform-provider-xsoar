# Components

## Account
XSOAR account that represents a tenant

## Host
Single server within a multi-tenant deployment
### Single Host
To configure a host (outside an HA Group) do the following:
- GET /host/download
- SCP installer to host server (address and keyfile provided in TF config)
- Via ssh execute the installer, including `-- -y` flag and elastic settings
- Once the installer completes the host will automatically connect to the main host and is viewable from the main api

### HA Group
Each HA Group is identified by an {id}. To configure a host do the following:
- GET /host/download/{id}
- SCP installer to host server (address and keyfile provided in TF config)
- Via ssh execute the installer, including `-- -y` flag
- Once the installer completes the host will automatically join the ha group and is viewable from the main api

## HA Group
Group of hosts that provide redundancy to each other.

### Building
- POST /ha-group/create
- Response contains an {id}
- POST /host/build/{id}
- This will take some time to return, should be done asynchronously 
- Once a 200 is returned the installer can be obtained

## Integration Instance
Instance of an integration within an account that is configured.