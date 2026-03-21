---
page_title: "2. Migrating to TF with Bulk Import and Config Bootstrapping"
subcategory: ""
description: |-
  Migrate existing deSEC domains and DNS records to Terraform management using list resources and bulk import.
---

# Migrating to TF with Bulk Import and Config Bootstrapping

If you have existing domains and DNS records in deSEC that were created outside of Terraform,
you can use [list resources](https://developer.hashicorp.com/terraform/language/import/bulk) to
discover them and generate Terraform configuration to bring them under management in bulk.

~> **Note:** Bulk import requires Terraform v1.12 or newer.

## Overview

The migration workflow has three steps:

1. **Query** — Use `terraform query` with a `.tfquery.hcl` file to discover existing resources.
2. **Generate** — Add the `-generate-config-out` flag to produce `resource` and `import` blocks.
3. **Apply** — Review the generated configuration, copy it into your main config, and run `terraform apply`.

## Step 1: Discover Existing Resources

Create a query file, for example `import.tfquery.hcl`, that lists the resources you want to import.

To discover all domains in your account:

```terraform
list "desec_domain" "all" {
  provider = desec
}
```

To discover all DNS records for a specific domain:

```terraform
list "desec_rrset" "all" {
  provider = desec
  config {
    domain = "example.com"
  }
}
```

You can combine multiple `list` blocks in the same file to import domains and their records
together.

```terraform
# Discover all domains in the account
list "desec_domain" "all" {
  provider = desec
}

# Discover all DNS RRsets for a domain
list "desec_rrset" "all" {
  provider = desec
  config {
    domain = "example.com"
  }
}
```

Run the query to preview what Terraform finds:

```shell
terraform query
```

## Step 2: Generate Configuration

Once you're satisfied with the query results, generate `resource` and `import` blocks:

```shell
terraform query -generate-config-out=generated.tf
```

This creates a `generated.tf` file containing:

- An `import` block for each discovered resource, with its identity.
- A `resource` block with the discovered attribute values.

Review the generated file. You may want to adjust resource names, reorganize blocks, or remove
resources you don't want to manage with Terraform.

## Step 3: Apply

Copy the `import` and `resource` blocks from `generated.tf` into your Terraform configuration,
then run:

```shell
terraform apply
```

Terraform imports each resource into state. From this point on, changes to these resources are
managed by Terraform.

-> **Tip:** You can remove the `import` blocks after a successful apply, or keep them as a
record of where the resources originated.

## Example: Import All Domains and Records

The following query file discovers all domains and all records for a specific domain:

```terraform
list "desec_domain" "all" {
  provider = desec
}

list "desec_rrset" "all" {
  provider = desec
  config {
    domain = "example.com"
  }
}
```

Run the full workflow:

```shell
terraform query -generate-config-out=generated.tf
# Review generated.tf, then:
terraform apply
```

## Next Steps

- To audit your account without importing, see [Auditing with List Resources](../guides/03-auditing-with-list-resources).
- See the [`desec_domain`](../list-resources/domain) and [`desec_rrset`](../list-resources/rrset) list resource references for all query options.
- Refer to Terraform's [bulk import documentation](https://developer.hashicorp.com/terraform/language/import/bulk) for more details on the `terraform query` workflow.
