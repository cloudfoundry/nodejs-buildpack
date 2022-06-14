

function ProcessSnapshotSender(agent) {
  this.agent = agent;

}
exports.ProcessSnapshotSender = ProcessSnapshotSender;


ProcessSnapshotSender.prototype.init = function() {
  var self = this;
  var libagentConnector = self.agent.libagentConnector;
  var processScanner = self.agent.processScanner;

  libagentConnector.on("autoProcessSnapshot", function() {
    processScanner.startAutoSnapshotIfPossible(function(err, processSnapshot) {
      if (err) {
        return;
      }

      libagentConnector.sendProcessSnapshot(processSnapshot);
    });
  });


  libagentConnector.on("processSnapshotRequest", function(processCallGraphReq) {
    processScanner.startManualSnapshot(processCallGraphReq, function(err, processSnapshot) {
      if (err) {
        return;
      }

      libagentConnector.sendProcessSnapshot(processSnapshot);
    });
  });
};


