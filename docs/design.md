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

### Approximate Approach
- POST to `/settings/integration/search`
- grab the `configurations` key
- search for the config that matches the name of the integration
- grab `configuration` array from returned object
```python
module_configuration = configuration["configuration"]
```
- create base module instance config
```python
module_instance = {
    'brand': configuration['name'],
    'category': configuration['category'],
    'configuration': configuration,
    'data': [],
    'enabled': "true",
    'engine': '',
    'id': '',
    'isIntegrationScript': is_byoi,
    'name': instance_name,
    'passwordProtected': False,
    'version': 0
}
```
- iterate through the module configuration options
- parse keys and values of config and add the values to the `module_instance`
- assign defaults as needed
- append the parameter to the `module_instance['data']` array
- send `module_instance` to `/settings/integration` as a PUT request

# License Requirements
