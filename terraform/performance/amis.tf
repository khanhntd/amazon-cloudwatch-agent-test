// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

variable "ami_family" {
  default = {
    debian = {
      login_user        = "ubuntu"
      install_package   = "amazon-cloudwatch-agent.deb"
      install_validator = "validator"
      temp_folder       = "/tmp"
      install_command   = "while sudo fuser /var/cache/apt/archives/lock /var/lib/apt/lists/lock /var/lib/dpkg/lock /var/lib/dpkg/lock-frontend; do echo 'Waiting for dpkg lock...' && sleep 1; done; echo 'No dpkg lock and install agent.' && sudo dpkg -i amazon-cloudwatch-agent.deb"
      start_command     = "sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a fetch-config -m ec2 -s -c file:%s"
      status_command    = "sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a status"
      connection_type   = "ssh"
      user_data         = ""
      wait_cloud_init   = "for i in {1..300}; do [ ! -f /var/lib/cloud/instance/boot-finished ] && echo 'Waiting for cloud-init...'$i && sleep 1 || break; done"
    }
    linux = {
      login_user        = "ec2-user"
      install_package   = "amazon-cloudwatch-agent.rpm"
      install_validator = "validator"
      temp_folder       = "/tmp"
      install_command   = "sudo rpm -Uvh amazon-cloudwatch-agent.rpm"
      start_command     = "sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a fetch-config -m ec2 -s -c file:%s"
      status_command    = "sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a status"
      connection_type   = "ssh"
      user_data         = ""
      wait_cloud_init   = "for i in {1..300}; do [ ! -f /var/lib/cloud/instance/boot-finished ] && echo 'Waiting for cloud-init...'$i && sleep 1 || break; done"
    }
    windows = {
      login_user        = "Administrator"
      install_package   = "amazon-cloudwatch-agent.msi"
      install_validator = "validator.exe"
      temp_folder       = "C:\\"
      install_command   = "msiexec /i C:\\amazon-cloudwatch-agent.msi"
      start_command     = "powershell \"& 'C:\\Program Files\\Amazon\\AmazonCloudWatchAgent\\amazon-cloudwatch-agent-ctl.ps1' -a fetch-config -m ec2 -s -c %s\""
      status_command    = "powershell \"& 'C:\\Program Files\\Amazon\\AmazonCloudWatchAgent\\amazon-cloudwatch-agent-ctl.ps1' -a status\""
      connection_type   = "ssh"
      wait_cloud_init   = " "
    }
  }
}
