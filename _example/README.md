# Example for EasyDNS Provider

Used for testing the provider against the Sandbox instance of the EasyDNS API.

This test is designed to be run 3 times:
- The first time it creates an entry (TXT `_acme-challenge.home.<dns_zone>` -- where dns_zone is the domain under test)
- The second time it updates that entry with a different set of text for the entry
- The third time it will delete the entry it created

## Setup
- Copy the `env-sample` file to `.env`
- Update the values for:
    - `EASYDNS_ZONE`: your domain, ie. mydomain.com
    - `EASYDNS_TOKEN`: at time of writing, this can be managed at: https://cp.easydns.com/manage/security/    
    - `EASYDNS_KEY`: at time of writing, this can be managed at: https://cp.easydns.com/manage/security/
    - `EASYDNS_URL`: set to the sandbox, and you probably want to test there

# Run the example
- From this directory, run:
```bash
    go run .
```

Each run will show you what is in the current zone (it will show all records).

At the bottom of the output (after the zone output) you will see text telling you what was done in the zone and a representation of a record showing what was added, updated, or deleted.
