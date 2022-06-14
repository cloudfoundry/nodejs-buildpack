

function MetricSender(agent) {
  this.agent = agent;
}
exports.MetricSender = MetricSender;



MetricSender.prototype.init = function() {
  var self = this;
  var libagentConnector = self.agent.libagentConnector;

  libagentConnector.on("connected", function() {

    // reset metrics
    self.agent.on('metricValue', function(metric, metricValue) {
      libagentConnector.reportMetric(metric.metricId, metricValue);
    });
  });

};
