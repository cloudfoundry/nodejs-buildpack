

function InstanceInfoSender(agent) {
  this.agent = agent;

}
exports.InstanceInfoSender = InstanceInfoSender;


InstanceInfoSender.prototype.init = function() {
  var self = this;
  var libagentConnector = self.agent.libagentConnector;
  var instanceTracker = self.agent.instanceTracker;

  libagentConnector.on("instanceTrackerConfig", function(instanceTrackerConfig) {
    instanceTracker.enabled = instanceTrackerConfig.enabled;
    instanceTracker.customTypes = instanceTrackerConfig.customTypes || [];

    if (instanceTracker.enabled && !instanceTracker.isRunning) {
      instanceTracker.isRunning = true;
      instanceTracker.startInstanceTracking(function(err, instanceCounts) {
        if (err) {
          return;
        }

        libagentConnector.sendInstanceTrackerInfo({instances: instanceCounts});
      });
    }
    else if (!instanceTracker.enabled && instanceTracker.isRunning) {
      instanceTracker.isRunning = false;
      instanceTracker.stopInstanceTracking();
    }
  });
};
