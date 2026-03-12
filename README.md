# Terraform / OpenTofu Provider for deSEC

[deSEC](https://desec.io) is a free, open-source DNS hosting service with a focus on security and privacy. This Terraform/OpenTofu provider lets you manage deSEC resources - DNS domains, record sets, API tokens, and token scoping policies - as infrastructure code.

## Using the provider

Install the provider by adding it to your `required_providers` block:

```hcl
terraform {
  required_providers {
    desec = {
      source  = "registry.terraform.io/timofurrer/desec"
      version = "~> 0.1"
    }
  }
}
```

Configure the provider with your deSEC API token. The token can also be
supplied via the `DESEC_API_TOKEN` environment variable:

```hcl
provider "desec" {
  api_token = "your-desec-api-token"
}
```

Create a DNS zone and add records:

```hcl
resource "desec_domain" "example" {
  name = "example.com"
}

resource "desec_record" "www" {
  domain  = desec_domain.example.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  records = ["1.2.3.4"]
}

data "desec_record" "nameservers" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "NS"
}

output "nameservers" {
  description = "The deSEC nameservers to enter at your domain registrar."
  value       = data.desec_record.nameservers.records
}
```

For the full list of resources, data sources, and configuration options see the
[provider documentation](https://registry.terraform.io/providers/timofurrer/desec/latest/docs).

Then commit the changes to `go.mod` and `go.sum`.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine.

To compile the provider, run `make playground`. This will build the provider and put the provider binary in the `./bin` directory.

Have a look at the `playground` folder to see how to use this development build.

To generate or update documentation, run `make generate`.

### Acceptance tests

By default, acceptance tests run against an in-memory fake deSEC API server.
No real account or credentials are needed, and each test gets a fresh isolated
server instance with no shared state:

```shell
make testacc
```

To run a specific test, use the `RUN` variable:

```shell
make testacc RUN=TestAccDomainResource
```

To run against the real deSEC API instead, set `DESEC_REAL_API=1` and provide
a valid API token via `DESEC_API_TOKEN`. An optional `DESEC_API_URL` override
is also supported (defaults to `https://desec.io/api/v1`):

```shell
DESEC_REAL_API=1 DESEC_API_TOKEN=your-token make testacc
```

**Note:** Running against the real API will create and delete actual resources
in your deSEC account and may trigger rate limits.
