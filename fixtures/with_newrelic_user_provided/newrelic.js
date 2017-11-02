/**
 * New Relic agent configuration.
 *
 * See lib/config.defaults.js in the agent distribution for a more complete
 * description of configuration variables and their potential values.
 */
exports.config = {
  app_name: ['My Application'],
  license_key: 'fake_new_relic_key1',
  logging: {
    level: 'info',
    filepath: 'stdout'
  },
  audit_log: {
    enabled: true,
    endpoints: []
  }
}
