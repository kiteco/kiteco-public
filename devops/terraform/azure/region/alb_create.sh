#!/bin/bash

# This script creates the prod and staging application gateways on azure using the azure cli

REGION=$1
SITE_ENV=$2
CERT_PATH=$3
CERT_PASS=$4

RESOURCE_GROUP=$(./tf_var.sh $REGION app-rg)
VPC_NAME=$(./tf_var.sh $REGION vpc-name)
SUBNET_PUBLIC=$(./tf_var.sh $REGION subnet-agw-public)
APP_GATEWAY_IP_ADDR=$(./tf_var.sh $REGION ip-${SITE_ENV}-app-gateway)
APP_GATEWAY_NAME=$(./tf_var.sh $REGION agw-${SITE_ENV}-name)

POOL_NAME="lbpool"
PROBE_NAME="${APP_GATEWAY_NAME}-probe"
LBSETTINGS_NAME="${APP_GATEWAY_NAME}-lbsettings"
LISTENER_NAME="${APP_GATEWAY_NAME}-listener"
SSL_CERT_NAME="${APP_GATEWAY_NAME}-cert"
SSL_PORT_NAME="${APP_GATEWAY_NAME}-port-ssl"

 set -e

if az network application-gateway list -g ${RESOURCE_GROUP} | grep -q "\"name\": \"${APP_GATEWAY_NAME}\""; then
    echo "gateway ${APP_GATEWAY_NAME} already exists"
else
    echo "[$(date +%R)] creating new application gateway ${APP_GATEWAY_NAME}"
    az network application-gateway create \
    --name "${APP_GATEWAY_NAME}" \
    --location "${REGION}" \
    --resource-group "${RESOURCE_GROUP}" \
    --tags deployments \
    --vnet-name "${VPC_NAME}" \
    --subnet "${SUBNET_PUBLIC}" \
    --capacity 2 \
    --sku Standard_Small \
    --public-ip-address "${APP_GATEWAY_IP_ADDR}"
fi


if az network application-gateway list -g ${RESOURCE_GROUP} | grep -q "\"name\": \"${SSL_CERT_NAME}\""; then
    echo "cert ${SSL_CERT_NAME} already exists"
else
    # new ssl cert
    echo "[$(date +%R)] creating new application gateway cert ${SSL_CERT_NAME}"
    az network application-gateway ssl-cert create \
    --name "${SSL_CERT_NAME}" \
    --cert-file $CERT_PATH \
    --cert-password $CERT_PASS \
    --gateway-name "${APP_GATEWAY_NAME}" \
    --resource-group "${RESOURCE_GROUP}"
fi

if az network application-gateway address-pool list -g ${RESOURCE_GROUP} --gateway-name ${APP_GATEWAY_NAME}| grep -q "\"name\": \"${POOL_NAME}\""; then
    echo "pool ${POOL_NAME} already exists"
else
    # new backend pool - CLI does not let you create empty pool so start it with a dummy value
    echo "[$(date +%R)] adding new pool"
    az network application-gateway address-pool create \
    --gateway-name "${APP_GATEWAY_NAME}" \
    --resource-group "${RESOURCE_GROUP}" \
    --name "${POOL_NAME}" \
    --servers 0
fi

if az network application-gateway list -g ${RESOURCE_GROUP} | grep -q "\"name\": \"${PROBE_NAME}\""; then
    echo "probe ${PROBE_NAME} already exists"
else
    # new probe configured to listen to /health (on default port 80)
    echo "[$(date +%R)] adding new probe"
    az network application-gateway probe create \
    --resource-group "${RESOURCE_GROUP}" \
    --gateway-name "${APP_GATEWAY_NAME}" \
    --name "${PROBE_NAME}" \
    --protocol "Http" \
    --host "127.0.0.1" \
    --path "/ready" \
    --interval 5 \
    --timeout 5 \
    --threshold 3
fi

if az network application-gateway list -g ${RESOURCE_GROUP} | grep -q "\"name\": \"${SSL_PORT_NAME}\""; then
    echo "frontend-port ${SSL_PORT_NAME} already exists"
else
# frontend port for ssl
echo "[$(date +%R)] adding new frontend port"
az network application-gateway frontend-port create \
--resource-group "${RESOURCE_GROUP}" \
--gateway-name "${APP_GATEWAY_NAME}" \
--name "${SSL_PORT_NAME}" \
--port 443
fi

if az network application-gateway list -g ${RESOURCE_GROUP} | grep -q "\"name\": \"${LISTENER_NAME}\""; then
    echo "listener ${LISTENER_NAME} already exists"
else
    # listener for port 443
    echo "[$(date +%R)] adding new http listener"
    az network application-gateway http-listener create \
    --resource-group "${RESOURCE_GROUP}" \
    --gateway-name "${APP_GATEWAY_NAME}" \
    --name "${LISTENER_NAME}" \
    --frontend-port "${SSL_PORT_NAME}" \
    --ssl-cert "${SSL_CERT_NAME}"
fi

if az network application-gateway list -g ${RESOURCE_GROUP} | grep -q "\"name\": \"${LBSETTINGS_NAME}\""; then
    echo "settings ${LBSETTINGS_NAME} already exists"
else
    # new http settings to use the above probe for port 80
    echo "[$(date +%R)] adding new http settings"
    az network application-gateway http-settings create \
    --resource-group "${RESOURCE_GROUP}" \
    --gateway-name "${APP_GATEWAY_NAME}" \
    --name "${LBSETTINGS_NAME}" \
    --timeout 30000 \
    --protocol "Http" \
    --port 80 \
    --probe "${PROBE_NAME}"
fi

# update the default rule rule1 to use the new pool and http settings
echo "[$(date +%R)] updating rule"
az network application-gateway rule update \
--resource-group "${RESOURCE_GROUP}" \
--gateway-name "${APP_GATEWAY_NAME}" \
--name "rule1" \
--address-pool "${POOL_NAME}" \
--http-settings "${LBSETTINGS_NAME}" \
--http-listener "${LISTENER_NAME}"

