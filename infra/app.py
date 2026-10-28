#!/usr/bin/env python3
"""
Reto técnico (Interseguro) — AWS CDK App
Cuenta: EYATECH (perfil `eyatech`), región us-east-1.

Uso (desde la raíz del repo, con Docker Desktop abierto):
  .\deploy.ps1                       despliega todo: pruebas, stack y frontend
  cd infra; cdk diff --profile eyatech
"""
import datetime

import aws_cdk as cdk

from stacks.reto_stack import RetoMatricesStack

app = cdk.App()
version = app.node.try_get_context("version") or datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%d-%H%M")

env = cdk.Environment(
    account=app.node.try_get_context("account") or None,
    region=app.node.try_get_context("region") or "us-east-1",
)
RetoMatricesStack(app, "RetoMatrices", version=version, env=env,
                  description="Reto técnico: API Go (QR) + API Node (estadísticas) + frontend Amplify")

cdk.Tags.of(app).add("Project", "RetoMatrices")
cdk.Tags.of(app).add("ManagedBy", "CDK")

app.synth()
