<p>Packages:</p>
<ul>
<li>
<a href="#proxy.shoot-auditlog-service.extensions.config.gardener.cloud%2fv1alpha1">proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1">proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the Certificate Shoot Service extension configuration.</p>
</p>
Resource Types:
<ul><li>
<a href="#proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration</a>
</li></ul>
<h3 id="proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration
</h3>
<p>
<p>Configuration contains information about the auditlog service configuration.</p>
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
proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1
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
<code>provider</code></br>
<em>
string
</em>
</td>
<td>
<p>Provider is the storage provider to store the auditlogs</p>
</td>
</tr>
<tr>
<td>
<code>providerConfig</code></br>
<em>
encoding/json.RawMessage
</em>
</td>
<td>
<em>(Optional)</em>
<p>ProviderConfig is the provider specific storage configuration</p>
</td>
</tr>
<tr>
<td>
<code>webhookConfiguration</code></br>
<em>
<a href="#proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.WebhookConfiguration">
WebhookConfiguration
</a>
</em>
</td>
<td>
<p>WebhookConfiguration holds the webhook specific configuration</p>
</td>
</tr>
</tbody>
</table>
<h3 id="proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.TLSConfiguration">TLSConfiguration
</h3>
<p>
(<em>Appears on:</em>
<a href="#proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.WebhookConfiguration">WebhookConfiguration</a>)
</p>
<p>
<p>TLSConfiguration contains the cert and key for tls</p>
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
<code>certFile</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>keyFile</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.WebhookConfiguration">WebhookConfiguration
</h3>
<p>
(<em>Appears on:</em>
<a href="#proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration</a>)
</p>
<p>
<p>WebhookConfiguration contains information about the proxy webhook endpoint</p>
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
<code>httpPort</code></br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>httpsPort</code></br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tls</code></br>
<em>
<a href="#proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1.TLSConfiguration">
TLSConfiguration
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<hr/>
