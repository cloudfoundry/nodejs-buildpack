var includeJavaProxy = process.env.npm_config_appd_include_java_proxy;
var globalInstall = process.env.npm_config_global;
if (includeJavaProxy && includeJavaProxy === "true") {
  var shell = require('shelljs');
  console.log('Install appdynamics-jre and appdynamics-proxy on-demand');
  var baseUrl,
    agentVersion = process.env.npm_package_version + '.0';

  if (process.env.APPD_STAGE === "true") {
    baseUrl = "https://packages-staging.corp.appdynamics.com";
  } else {
    baseUrl = "https://packages.appdynamics.com";
  }

  var appdynamicsJreDepPath = baseUrl + "/nodejs/" + agentVersion + "/appdynamics-jre.tgz",
    appdynamicsProxyDepPath = baseUrl + "/nodejs/" + agentVersion + "/appdynamics-proxy.tgz";

  console.log('Computed path for jre dependency ' + appdynamicsJreDepPath);
  console.log('Computed path for proxy dependency ' + appdynamicsProxyDepPath);
  var appdynamicsJreInstallResult = shell.exec("npm install " + appdynamicsJreDepPath + (globalInstall == "true" ? " --global" : ""));
  var appdynamicsProxyInstallResult = shell.exec("npm install " + appdynamicsProxyDepPath + (globalInstall == "true" ? " --global" : ""));
  if (appdynamicsJreInstallResult.code != 0) {
    console.log('Installation of the appdynamics-jre failed');
    process.exit(1);
  }
  if (appdynamicsProxyInstallResult.code != 0) {
    console.log('Installation of appdynamics-proxy failed');
    process.exit(1);
  }
} else {
  console.log('Install for libagent mode only');
  return;
}