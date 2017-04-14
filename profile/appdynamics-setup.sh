FILTER="appdynamics\|app-dynamics"

if [ `echo $VCAP_SERVICES | grep -c $FILTER ` -gt 0 ];
then
  key="appdynamics"
  APPDYNAMICS_CONTROLLER_HOST_NAME=$(echo "${VCAP_SERVICES-}" | jq -r '.['\""$key"\"'][0] | .credentials | .["host-name"] ')
  APPDYNAMICS_CONTROLLER_PORT=$(echo "${VCAP_SERVICES-}" | jq -r '.['\""$key"\"'][0] | .credentials | .port ')
  APPDYNAMICS_AGENT_ACCOUNT_NAME=$(echo "${VCAP_SERVICES-}" | jq -r '.['\""$key"\"'][0] | .credentials | .["account-name"] ')

  APPDYNAMICS_CONTROLLER_SSL_ENABLED=$(echo "${VCAP_SERVICES-}" | jq -r '.['\""$key"\"'][0] | .credentials | .["ssl-enabled"] ')
  APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY=$(echo "${VCAP_SERVICES-}" | jq -r '.['\""$key"\"'][0] | .credentials | .["account-access-key"] ')
  APPDYNAMICS_AGENT_APPLICATION_NAME=$(echo "${VCAP_APPLICATION-}" | jq -r .application_name)
  APPDYNAMICS_AGENT_TIER_NAME=$(echo "${VCAP_APPLICATION-}" | jq -r .application_name)
  APPDYNAMICS_AGENT_NODE_NAME_PREFIX=$(echo "${APPDYNAMICS_AGENT_TIER_NAME}")

  if [ ! -z "${APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY-}" ];
  then
    if [ ! -z "${APPDYNAMICS_CONTROLLER_HOST_NAME-}" ]; then
      export APPDYNAMICS_CONTROLLER_HOST_NAME=$APPDYNAMICS_CONTROLLER_HOST_NAME
    fi
    if [ ! -z "${APPDYNAMICS_CONTROLLER_PORT-}" ]; then
      export APPDYNAMICS_CONTROLLER_PORT=$APPDYNAMICS_CONTROLLER_PORT
    fi
    if [ ! -z "${APPDYNAMICS_CONTROLLER_SSL_ENABLED-}" ]; then
      export APPDYNAMICS_CONTROLLER_SSL_ENABLED=$APPDYNAMICS_CONTROLLER_SSL_ENABLED
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_ACCOUNT_NAME-}" ]; then
      export APPDYNAMICS_AGENT_ACCOUNT_NAME=$APPDYNAMICS_AGENT_ACCOUNT_NAME
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY-}" ]; then
      export APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY=$APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_APPLICATION_NAME-}" ]; then
      export APPDYNAMICS_AGENT_APPLICATION_NAME=$APPDYNAMICS_AGENT_APPLICATION_NAME
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_TIER_NAME-}" ]; then
      export APPDYNAMICS_AGENT_TIER_NAME=$APPDYNAMICS_AGENT_TIER_NAME
    fi
    if [ ! -z "${APPDYNAMICS_AGENT_NODE_NAME_PREFIX}" ]; then
      export APPDYNAMICS_AGENT_NODE_NAME=$APPDYNAMICS_AGENT_NODE_NAME_PREFIX:\$CF_INSTANCE_INDEX
    fi
  fi
fi
