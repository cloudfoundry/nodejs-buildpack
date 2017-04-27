list_dependencies() {
  local build_dir="$1"

  cd "$build_dir"
  if $YARN; then
    echo ""
    (yarn list --depth=0 || true) 2>/dev/null
    echo ""
  else
    (npm ls --depth=0 | tail -n +2 || true) 2>/dev/null
  fi
}

run_if_present() {
  local script_name=${1:-}
  local has_script=$(jq -r ".scripts[\"$script_name\"] // \"\"" < "$BUILD_DIR/package.json")
  if [ -n "$has_script" ]; then
    if $YARN; then
      echo "Running $script_name (yarn)"
      yarn run "$script_name"
    else
      echo "Running $script_name"
      npm run "$script_name" --if-present
    fi
  fi
}

yarn_node_modules() {
  local build_dir=${1:-}
  echo "Installing node modules (yarn.lock)"
  cd "$build_dir"

  local mirror_dir="$build_dir/npm-packages-offline-cache"

  if [ -d "$mirror_dir" ]; then
    echo "Found yarn mirror directory $mirror_dir"
    echo "Running yarn in offline mode"
    yarn config set yarn-offline-mirror "$mirror_dir"
    run_yarn $build_dir --offline
  else
    echo "Running yarn in online mode"
    echo "To run yarn in offline mode, see: https://yarnpkg.com/blog/2016/11/24/offline-mirror"
    run_yarn $build_dir
  fi
}

run_yarn() {
    local build_dir=${1:-}
    local offline_flag=${2:-}

    # if there are native modules, yarn will use node-gyp to rebuild them
    # setting npm_config_nodedir tells npm-gyp to use the node header files from
    # that directory. Otherwise, node-gyp will try to download the headers.

    export npm_config_nodedir=$NODE_HOME
    yarn install $offline_flag --pure-lockfile --ignore-engines --cache-folder $build_dir/.cache/yarn 2>&1
    unset npm_config_nodedir

    # according to docs: "Verifies that versions of the package dependencies in the current project’s package.json matches that of yarn’s lock file."
    # however, appears to also check for the presence of deps in node_modules, so must be run after install
    if $(yarn check $offline_flag 1>/dev/null); then
      echo "yarn.lock and package.json match"
    else
      echo "yarn.lock is outdated"
      warning "yarn.lock is outdated." "run \`yarn install\`, commit the updated \`yarn.lock\`, and redeploy"
    fi
}

npm_node_modules() {
  local build_dir=${1:-}

  if [ -e $build_dir/package.json ]; then
    cd $build_dir

    if [ -e $build_dir/npm-shrinkwrap.json ]; then
      echo "Installing node modules (package.json + shrinkwrap)"
    else
      echo "Installing node modules (package.json)"
    fi
    npm install --unsafe-perm --userconfig $build_dir/.npmrc --cache $build_dir/.npm 2>&1
  else
    echo "Skipping (no package.json)"
  fi
}

npm_rebuild() {
  local build_dir=${1:-}

  if [ -e $build_dir/package.json ]; then
    cd $build_dir
    echo "Rebuilding any native modules"
    npm rebuild --nodedir=$NODE_HOME 2>&1
    if [ -e $build_dir/npm-shrinkwrap.json ]; then
      echo "Installing any new modules (package.json + shrinkwrap)"
    else
      echo "Installing any new modules (package.json)"
    fi
    npm install --unsafe-perm --userconfig $build_dir/.npmrc 2>&1
  else
    echo "Skipping (no package.json)"
  fi
}
