#!/bin/sh

set -x

EAST_US_REGIONS="GEO-AN US US-AL US-AR US-CT US-DC US-DE US-FL US-GA US-IA US-IL US-IN US-KS US-KY US-LA US-MA US-MD US-ME US-MI US-MN US-MO US-MS US-NC US-NH US-NJ US-NY US-OH US-OK US-PA US-RI US-SC US-TN US-TX US-VA US-VT US-WI US-WV"
WEST_US_REGIONS="WORLD GEO-AP GEO-NA GEO-SA US-AK US-AZ US-CA US-CO US-HI US-ID US-MT US-ND US-NE US-NM US-NV US-OR US-SD US-UT US-WA US-WY"
WEST_EU_REGIONS="GEO-EU GEO-AF GEO-AS"


EAST_US_AGW_STAGING="/subscriptions/XXXXXXX/resourceGroups/prod-eastus-0/providers/Microsoft.Network/publicIPAddresses/staging-app-gateway-ip"
WEST_US_AGW_STAGING="/subscriptions/XXXXXXX/resourceGroups/prod-westus2-0/providers/Microsoft.Network/publicIPAddresses/staging-app-gateway-ip"
WEST_EU_AGW_STAGING="/subscriptions/XXXXXXX/resourceGroups/prod-westeurope-0/providers/Microsoft.Network/publicIPAddresses/staging-app-gateway-ip"

EAST_US_AGW_PROD="/subscriptions/XXXXXXX/resourceGroups/prod-eastus-0/providers/Microsoft.Network/publicIPAddresses/prod-app-gateway-ip"
WEST_US_AGW_PROD="/subscriptions/XXXXXXX/resourceGroups/prod-westus2-0/providers/Microsoft.Network/publicIPAddresses/prod-app-gateway-ip"
WEST_EU_AGW_PROD="/subscriptions/XXXXXXX/resourceGroups/prod-westeurope-0/providers/Microsoft.Network/publicIPAddresses/prod-app-gateway-ip"


az network traffic-manager profile create \
-g "traffic-manager-public" \
--name "kite-staging" \
--routing-method "Geographic" \
--unique-dns-name "kite-staging" \
--monitor-port "443" \
--monitor-protocol "https"

az network traffic-manager endpoint create \
-g "traffic-manager-public" \
--name "east-us" \
--profile-name "kite-staging" \
--type "azureEndpoints" \
--geo-mapping $EAST_US_REGIONS \
--target-resource-id "${EAST_US_AGW_STAGING}"

az network traffic-manager endpoint create \
-g "traffic-manager-public" \
--name "west-us" \
--profile-name "kite-staging" \
--type "azureEndpoints" \
--geo-mapping $WEST_US_REGIONS \
--target-resource-id "${WEST_US_AGW_STAGING}"

az network traffic-manager endpoint create \
-g "traffic-manager-public" \
--name "west-eu" \
--profile-name "kite-staging" \
--type "azureEndpoints" \
--geo-mapping $WEST_EU_REGIONS \
--target-resource-id "${WEST_EU_AGW_STAGING}"


az network traffic-manager profile create \
-g "traffic-manager-public" \
--name "kite-prod" \
--routing-method "Geographic" \
--unique-dns-name "kite-prod" \
--monitor-port "443" \
--monitor-protocol "https"

az network traffic-manager endpoint create \
-g "traffic-manager-public" \
--name "east-us" \
--profile-name "kite-prod" \
--type "azureEndpoints" \
--geo-mapping $EAST_US_REGIONS \
--target-resource-id "${EAST_US_AGW_PROD}"

az network traffic-manager endpoint create \
-g "traffic-manager-public" \
--name "west-us" \
--profile-name "kite-prod" \
--type "azureEndpoints" \
--geo-mapping $WEST_US_REGIONS \
--target-resource-id "${WEST_US_AGW_PROD}"

az network traffic-manager endpoint create \
-g "traffic-manager-public" \
--name "west-eu" \
--profile-name "kite-prod" \
--type "azureEndpoints" \
--geo-mapping $WEST_EU_REGIONS \
--target-resource-id "${WEST_EU_AGW_PROD}"