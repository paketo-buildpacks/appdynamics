# `docker.io/paketobuildpacks/appdynamics`
The Paketo Buildpack for AppDynamics is a Cloud Native Buildpack that contributes the [AppDynamics][n] Agent and configures it to
connect to the service.

[a]: https://www.appdynamics.com

## Behavior
This buildpack will participate if all the following conditions are met

* A binding exists with `type` of `AppDynamics`

The buildpack will do the following for Java applications:

* Contributes a Java agent to a layer and configures `$JAVA_TOOL_OPTIONS` to use it
  * Contributes a default `app-agent-config.xml`, `custom-activity-correlation.xml`, and `log4j2.xml`
* Contribute external configuration if available
* Transforms the contents of the binding secret to environment variables with the pattern `APPDYNAMICS_<KEY>=<VALUE>`

The buildpack will do the following for NodeJS applications:

* Contributes a NodeJS agent to a layer and configures `$NODE_MODULES` to use it
* If main module does not already require `appdynamics` module, prepends the main module with `require('appdynamics');`
* Transforms the contents of the binding secret to environment variables with the pattern `APPDYNAMICS_<KEY>=<VALUE>`

The buildpack will do the following for PHP applications:

* Contributes a PHP agent to a layer and configures `$PHP_INI_SCAN_DIR` to use it
* Transforms the contents of the binding secret to environment variables with the pattern `APPDYNAMICS_<KEY>=<VALUE>`

## Configuration
| Environment Variable | Description
| -------------------- | -----------
| `$APPDYNAMICS_AGENT_APPLICATION_NAME` | Configure the AppDynamics application name
| `$APPDYNAMICS_AGENT_NODE_NAME` | Configure the AppDynamics node name
| `$APPDYNAMICS_AGENT_TIER_NAME` | Configure the AppDynamics tier name
| `$BP_APPD_EXT_CONF_SHA256` | Configure the SHA256 hash of the external AppDynamics configuration archive
| `$BP_APPD_EXT_CONF_STRIP` | Configure the number of directory components to strip from the external AppDynamics configuration archive. Defaults to `0`.
| `$BP_APPD_EXT_CONF_URI` | Configure the download location of the external AppDynamics configuration
| `$BP_APPD_EXT_CONF_VERSION` | Configure the version of the external AppDynamics configuration

## Bindings
The buildpack optionally accepts the following bindings:

### Type: `dependency-mapping`
|Key                   | Value   | Description
|----------------------|---------|------------
|`<dependency-digest>` | `<uri>` | If needed, the buildpack will fetch the dependency with digest `<dependency-digest>` from `<uri>`

## ARM64 Support

This buildpack supports running on ARM64, however, not for all language families. Presently, it supports the Java Agent on ARM64.

## License

This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
