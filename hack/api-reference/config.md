<p>Packages:</p>
<ul>
<li>
<a href="#shoot-auditlog-service.extensions.config.gardener.cloud%2fv1alpha1">shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1">shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the Certificate Shoot Service extension configuration.</p>
</p>
Resource Types:
<ul><li>
<a href="#shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration</a>
</li></ul>
<h3 id="shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration
</h3>
<p>
<p>Configuration contains information about the auditlog service configuration.
todo: add support for multiple providers</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Configuration</code></td>
</tr>
<tr>
<td>
<code>healthCheckConfig</code></br>
<em>
<a href="https://github.com/gardener/gardener-extensions/pkg/controller/healthcheck/config">
github.com/gardener/gardener-extensions/pkg/controller/healthcheck/config/v1alpha1.HealthCheckConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>HealthCheckConfig is the config for the health check controller</p>
</td>
</tr>
</tbody>
</table>
<hr/>
