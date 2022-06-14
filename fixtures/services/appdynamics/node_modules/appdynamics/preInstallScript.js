var nodeVersionParts = process.version.split(".");
if(parseInt(nodeVersionParts[0].substring(1), 10) < 8) {
  console.log("This version of Appdynamics agent only supports Node.js 8.0 and above. \n"
            + "For older versions of Node.js please use the AppDynamics agent 4.5.11 by installing with 'npm install appdynamics@4.5.11'");
  process.exit(1);
}
