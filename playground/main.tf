# Copyright Timo Furrer 2026
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    desec = {
      source = "timofurrer/desec"
    }
  }
}

# NOTE: set TF_CLI_CONFIG_FILE=$(pwd)/.terraformrc

provider "desec" {
}

resource "desec_domain" "example" {
  name = "example.ch"
}
