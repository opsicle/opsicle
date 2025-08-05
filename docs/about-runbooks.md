# What's a Runbook?

A runbook is a structured set of instructions that describes how to perform a specific operational task in a system. It can range from simple procedures like restarting a service to complex workflows involving database failover, access provisioning, or multi-step deployments. Traditionally maintained as documents or wikis, modern runbooks are often codified in machine-executable formats to enable automation, version control, and integration into CI/CD pipelines.

The primary value of a runbook lies in consistency and reliability. By clearly defining each step, runbooks reduce the risk of human error, ensure repeatability, and enable less experienced engineers to safely perform tasks that would otherwise require deep system knowledge. They also serve as a critical tool during high-pressure scenarios like incident response, where having a clear, validated procedure can dramatically reduce time to resolution.

In systems like Opsicle, runbooks go beyond documentation. They are live, executable automation units. Written in YAML and triggered via our CLI tool, API, or web application, they can include safeguards like approval gates, pre-defined execution environments, and comprehensive logging. This transforms runbooks from static references (*do X, then do Y, and finally do Z*) into dynamic components of system operations (*enter in variables and click "Run"*), enabling scalable, secure, and auditable infrastructure management.

## Why use a runbook automation platform?

### Consistency

Runbooks eliminate reliance on tribal knowledge by providing a single source of truth for how operational tasks are performed. Instead of engineers improvising or asking around for the "right way" to do something, runbooks that define the process explicitly can be written by the people most apt for writing it: the developers themselves. This ensures that no matter who runs the task or when it’s done, it happens the same way every single time, as intended by its creators.

### Efficiency

With a well-defined runbook, engineers don’t waste time figuring out next steps or debugging ad hoc scripts. Whether it's deploying to production or recovering from an outage, having a runbook reduces the time it takes to go from identification to resolution. This is especially critical during incidents, when every minute counts.

### Security

Runbooks help prevent costly mistakes by treating security guardrails as a first-class member in the world of operations. Unlike a support engineer being granted privileges and being trusted to handle the task, runbooks can include built-in code validation, pre-defined permissions, approval mechanisms, and restricted access to ensure sensitive tasks like database `UPDATE`s or credential rotations aren’t accidentally or maliciously executed. Logs and metadata from every execution also provide full traceability.

### Productivity

Runbooks are a foundation for automation. What starts as a checklist or manual process can gradually be transformed into a fully automated workflow, triggered by events, schedules, or API calls. This enables teams to scale operations without scaling headcount, reducing operational toil so that engineers can focus on strategic work.

## Common use-cases of runbooks

### On-Call Incident Response (e.g., Service Outage)

When a production service goes down, time and clarity are critical. An on-call engineer might be alerted at 3 AM with a vague symptom like "*The API is slowww,*" or "*We're getting `5xx` errors from the onboarding service,*" In this scenario, a runbook helps by guiding the responder through:

1. Checking dashboards or logs (with predefined Grafana/Datadog links)
2. Validating that the issue isn’t caused by a known dependency (e.g., a third-party outage)
3. Running diagnostic commands or automations (e.g., checking database locks, restarting specific pods)
4. Notifying stakeholders and creating a postmortem ticket

With a runbook, even your L1 support engineer can confidently follow the exact steps to diagnose and mitigate the issue without unnecessary escalations. It also ensures that each incident response is logged, auditable, and consistent across shifts.

### Automate Compute Workload Scaling

Most workloads have predictable usage patterns especially when it comes to internal tools or batch processing systems that are idle outside business hours. Instead of spending thousands on compute 24/7, we could use a runbook to automate the scaling down of your infrastructure on weekends and subsequent scaling up before the work week begins.

A typical runbook for this might:

1. Drain non-critical nodes: Safely cordon and drain nodes running dev/staging workloads or batch jobs.
2. Scale down autoscaling groups / node pools: Reduce the min/max/desired instance counts via your cloud provider or Kubernetes autoscaler.
3. Suspend non-essential services: Stop auxiliary workloads (e.g., staging CI runners, test environments) to free up resources.
4. Notify stakeholders: Send a Slack or email update when the scale-down is complete, including a summary of what was scaled down and how much cost is saved.
5. Set an auto-reversal trigger: Schedule or conditionally run a companion runbook Monday morning to scale back up and ensure readiness for the following week's grind.

This reduces cloud spend while maintaining operational hygiene. By codifying such an operation in a runbook, you can ensure the scale-in/out is repeatable, auditable, and safe (e.g., no production workloads are impacted) - all without relying on Platform Engineering expertise

### Rotating Expired or Leaked Credentials

Credential rotations (eg. AWS access keys, database passwords, webhook secrets *et cetera*) are generally high-risk, low-frequency tasks that are easy to get wrong simply because it's usually done manually, and it's not done often. A runbook written by the original implementer can guide or automate this process for future maintainers end-to-end:

1. Generating a new credential using secure tooling
2. Updating secrets in the appropriate store (e.g., Hashicorp Vault, AWS SecretsManager, Kubernetes Secret)
3. Rolling out deployments that depend on the updated secret
4. Verifying that services using the secret are healthy post-rotation
5. Cleaning up old credentials and confirming they're revoked

This ensures that rotations are safe, atomic, and doesn't break downstream services. Having an auditable runbook also satisfies security and compliance requirements.

### Provisioning Access for a New Engineer

Giving a new engineer access to infrastructure, dashboards, CI/CD pipelines, and internal systems usually involve multiple teams and manual steps. A runbook standardises and automates the onboarding process by:

1. Creation of accounts in identity providers (e.g., Azure AD, Okta)
2. Assignment of roles or group memberships
3. Requesting of approvals from team leads via Slack or Jira
4. Granting temporary access tokens or VPN credentials
5. Setting up monitoring and expiry reminders for time-limited access

This ensures that steps are not skipped, access is securely provisioned, and the process is auditable. It can also make off-boarding just as reliable by reversing the same steps when someone leaves.

### Tenant On/Off-boarding in Multitenant Systems

For SaaS platforms, on-and-off-boarding a customer typically involves:

1. Creation of (or archiving/deletion of) tenant data
2. Granting (or revoking) user access
3. Setting up (or cleaning up) infrastructure resources like databases, buckets, and queues
4. Updating of finance and support systems
5. Audits for residual access or data

Codified runbooks make tenant lifecycle operations safer and reduce risk of partial cleanup or data leaks.

### Manual Approvals for Sensitive Operations

Certain operations like targetted deletion of production data or pausing of payment systems should never happen without human approval. A runbook helps by allowing teams to:

1. Define the operation in code
2. Trigger an approval flows via the organisation's communication channels
3. Proceed or abort based on a pre-defined approver's decision
4. Log who approved it and when
5. Collate logs generated by the script/code

This allows real humans on your team to make decisions without worrying about a privileged engineer hitting delete on a wrong part of the production data because they had a late night yesterday.
