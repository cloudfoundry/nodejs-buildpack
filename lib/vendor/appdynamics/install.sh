#!/usr/bin/env bash

set -e
set -o pipefail
set -o nounset

SCRIPT_PATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

BUILD_DIR=$1
BP_DIR=$SCRIPT_PATH/../../..

. $SCRIPT_PATH/../../json.sh

if [ `echo $VCAP_SERVICES | grep -c "appdynamics" ` -gt 0 ];
then
  key="appdynamics"
  if [ `echo $VCAP_SERVICES | grep -c "user-provided" ` -gt 0 ];
  then
    key="user-provided"
  fi
  APPDYNAMICS_CONTROLLER_HOST_NAME=$(echo "${VCAP_SERVICES-}" | $JQ --raw-output '.['\""$key"\"'][0] | .credentials | .["host-name"] ')
  APPDYNAMICS_CONTROLLER_PORT=$(echo "${VCAP_SERVICES-}" | $JQ --raw-output '.['\""$key"\"'][0] | .credentials | .port ')
  APPDYNAMICS_AGENT_ACCOUNT_NAME=$(echo "${VCAP_SERVICES-}" | $JQ --raw-output '.['\""$key"\"'][0] | .credentials | .["account-name"] ')
  APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY=$(echo $VCAP_SERVICES | $JQ --raw-output '.['\""$key"\"'][0] | .credentials | .["account-access-key"] ')

  APPDYNAMICS_AGENT_APPLICATION_NAME=$(echo "${VCAP_APPLICATION-}" | $JQ --raw-output .application_name)
  APPDYNAMICS_AGENT_TIER_NAME=$(echo "${VCAP_APPLICATION-}" | $JQ --raw-output .application_name)
  APPDYNAMICS_AGENT_NODE_NAME=$(echo "${VCAP_APPLICATION-}" | $JQ --raw-output .application_name)

  if [ ! -z "${APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY-}" ];
  then
    mkdir -p $BUILD_DIR/.profile.d
    SETUP_APPDYNAMICS=$BUILD_DIR/.profile.d/appdynamics-setup.sh

    if [ ! -z "${APPDYNAMICS_CONTROLLER_HOST_NAME-}" ]; then
      echo "export APPDYNAMICS_CONTROLLER_HOST_NAME=$APPDYNAMICS_CONTROLLER_HOST_NAME" >> $SETUP_APPDYNAMICS
    fi
    if [ ! -z "${APPDYNAMICS_CONTROLLER_PORT-}" ]; then
      echo "export APPDYNAMICS_CONTROLLER_PORT=$APPDYNAMICS_CONTROLLER_PORT" >> $SETUP_APPDYNAMICS
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_ACCOUNT_NAME-}" ]; then
      echo "export APPDYNAMICS_AGENT_ACCOUNT_NAME=$APPDYNAMICS_AGENT_ACCOUNT_NAME" >> $SETUP_APPDYNAMICS
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY-}" ]; then
      echo "export APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY=$APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY" >> $SETUP_APPDYNAMICS
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_APPLICATION_NAME-}" ]; then
      echo "export APPDYNAMICS_AGENT_APPLICATION_NAME=$APPDYNAMICS_AGENT_APPLICATION_NAME" >> $SETUP_APPDYNAMICS
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_TIER_NAME-}" ]; then
      echo "export APPDYNAMICS_AGENT_TIER_NAME=$APPDYNAMICS_AGENT_TIER_NAME" >> $SETUP_APPDYNAMICS
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_NODE_NAME-}" ]; then
      echo "export APPDYNAMICS_AGENT_NODE_NAME=$APPDYNAMICS_AGENT_NODE_NAME" >> $SETUP_APPDYNAMICS
    fi
  fi
fi
