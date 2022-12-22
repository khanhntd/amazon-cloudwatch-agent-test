// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

resource "aws_security_group" "ecs_security_group" {
  name   = module.common.vpc_security_group
}