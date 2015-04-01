warn_semver_range() {
  local semver_range=$1
  if [ "$semver_range" == "" ]; then
    warning "Node version not specified in package.json"
  elif [ "$semver_range" == "*" ]; then
    warning "Avoid semver ranges like '*' in engines.node"
  elif [ ${semver_range:0:1} == ">" ]; then
    warning "Avoid semver ranges starting with '>' in engines.node"
  fi
}

warn_node_modules() {
  local modules_source=$1
  if [ "$modules_source" == "prebuilt" ]; then
    warning "Avoid checking node_modules into source control"
  elif [ "$modules_source" == "" ]; then
    warning "No package.json found"
  fi
}

warn_start() {
  local start_method=$1
  if [ "$start_method" == "" ]; then
    warning "No Procfile, package.json start script, or server.js file found"
  fi
}

warn_old_npm() {
  local npm_version=$1
  if [ "${npm_version:0:1}" -lt "2" ]; then
    local latest_npm=$(curl --silent --get https://semver.herokuapp.com/npm/stable)
    warning "This version of npm ($npm_version) has several known issues - consider upgrading to the latest release ($latest_npm)"
  fi
}
