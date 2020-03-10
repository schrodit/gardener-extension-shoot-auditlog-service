<p>Packages:</p>
<ul>
<li>
<a href="#service.auditlog.extensions.config.gardener.cloud%2fv1alpha1">service.auditlog.extensions.config.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="service.auditlog.extensions.config.gardener.cloud/v1alpha1">service.auditlog.extensions.config.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the Certificate Shoot Service extension configuration.</p>
</p>
Resource Types:
<ul><li>
<a href="#service.auditlog.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration</a>
</li></ul>
<h3 id="service.auditlog.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration
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
service.auditlog.extensions.config.gardener.cloud/v1alpha1
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
<code>backendProvider</code></br>
<em>
string
</em>
</td>
<td>
<p>BackendProvider specifies the provider for the audit log proxy where the logs are persisted</p>
</td>
</tr>
<tr>
<td>
<code>backendProviderConfig</code></br>
<em>
encoding/json.RawMessage
</em>
</td>
<td>
<p>BackendProviderConfig is the backend provider specific configuration</p>
</td>
</tr>
<tr>
<td>
<code>policy</code></br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/runtime#RawExtension">
k8s.io/apimachinery/pkg/runtime.RawExtension
</a>
</em>
</td>
<td>
<p>Policy is the raw audit log policy.
Be aware that k8s clusters &lt;=1.11 do not support &ldquo;audit.k8s.io/v1&rdquo;</p>
</td>
</tr>
</tbody>
</table>
<hr/>
