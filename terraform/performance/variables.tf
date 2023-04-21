variable "region" {
  type    = string
  default = "us-west-2"
}

variable "ec2_instance_type" {
  type    = string
  default = "t3a.xlarge"
}

variable "ssh_key_name" {
  type    = string
  default = ""
}

variable "ami" {
  type    = string
  default = "cloudwatch-agent-integration-test-al2*"
}

variable "ssh_key_value" {
  type    = string
  default = ""
}

variable "arc" {
  type    = string
  default = ""
}

variable "s3_bucket" {
  type    = string
  default = "integration-test-cwagent"
}

variable "test_dir" {
  type    = string
  default = "../../test/stress/statsd"
}

variable "cwa_github_sha" {
  type    = string
  default = "0086f4cce10fa27f5604602778bf17b113c7bdf1"
}

variable "cwa_github_sha_date" {
  type    = string
  default = ""
}
variable "values_per_minute" {
  type    = number
  default = 10
}

variable "family" {
  type    = string
  default = "linux"

  validation {
    condition     = contains(["windows", "mac", "linux"], var.family)
    error_message = "Valid values for family are (windows, mac, linux)."
  }
}
