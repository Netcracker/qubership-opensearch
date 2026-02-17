# Migrating OpenSearch 1.x Indices

When upgrading to OpenSearch 3.x, indices created in OpenSearch 1.x must be migrated. This guide explains how to check for legacy indices and perform the migration.

## Quick Start

### Check for Legacy Indices

Run a dry-run to identify indices that need migration:

```bash
# Exec into curator pod
kubectl exec -it $(kubectl get pods -n opensearch -l component=opensearch-curator -o jsonpath='{.items[0].metadata.name}') -n opensearch -- bash

# Check for legacy indices (dry run)
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --dry-run
```

The script will list all OpenSearch 1.x indices and show:
- Index names and sizes
- Required disk space
- Migration steps that would be performed

### Run Migration

Once you've reviewed the dry-run output, execute the migration:

```bash
# Run migration
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py
```

## Migration Approaches

### Option 1: Manual Migration (Recommended for Production)

Manual migration gives you full control over the process:

```bash
# Step 1: Exec into curator pod
kubectl exec -it $(kubectl get pods -n opensearch -l component=opensearch-curator -o jsonpath='{.items[0].metadata.name}') -n opensearch -- bash

# Step 2: Dry run to check what will be migrated
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --dry-run

# Step 3: Run migration
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py
```

### Option 2: Automatic Migration During Upgrade

Enable automatic migration in your values file:

```yaml
# values.yaml
migration:
  enabled: true  # Automatically migrate during Helm upgrade
```

Then upgrade to OpenSearch 3.x:

```bash
helm upgrade opensearch ./opensearch-service -f values.yaml -n opensearch --wait
```

**Important**: The pre-deploy job always runs for OpenSearch 3.x upgrades. By default (`migration.enabled: false`), it only checks for legacy indices and fails the upgrade if found during 2.x to 3.x upgrade. Set `enabled: true` to perform automatic migration.

## Pre-Deploy Check Behavior

When upgrading to OpenSearch 3.x with `migration.enabled: false` (default):

- **No legacy indices**: Upgrade proceeds normally
- **Legacy indices but not 2.x to 3.x upgrade**: Warning only, upgrade proceeds
- **Legacy indices during 2.x to 3.x upgrade**: Upgrade fails with message:

```
ERROR: Legacy OpenSearch 1.x indices detected during 2.x to 3.x upgrade.
Please install the latest OpenSearch 2.x version and perform manual migration first.
See documentation: docs/public/migration-indices.md
```

## Command Options

```bash
# Dry run - show what will be migrated without making changes
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --dry-run

# Skip backup creation (not recommended for production)
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --skip-backup

# Skip disk space validation (not recommended)
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --skip-space-check

# Skip security configuration backup
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --skip-security-backup

# Skip DBaaS user restoration
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --skip-dbaas-restore

# Combine multiple options
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --skip-backup --skip-space-check
```

## Expected Duration

**Note**: The migration script automatically creates a backup before starting (if curator is enabled). This adds additional time to the overall process.

Migration time depends on index size:

| Index Size | Estimated Time |
|------------|----------------|
| Less than 1GB | 1-5 minutes |
| 1-10GB | 5-30 minutes |
| 10-100GB | 30-180 minutes |
| 100GB+ | Several hours |

**Additional Time**:
- Backup creation: 5-30 minutes (depends on cluster size)
- Security reinitialization: 1-2 minutes
- DBaaS user restoration: 1-5 minutes

**Note**: The script processes indices sequentially (one at a time) to minimize cluster load.

## Environment Variables

The curator pod already has all required environment variables configured. The migration script uses these same variables:

| Variable | Description | Source |
|----------|-------------|--------|
| `ES_HOST` | OpenSearch endpoint (e.g., `opensearch-internal:9200`) | Configured from Helm values |
| `ES_USERNAME` | Admin username | From opensearch-secret |
| `ES_PASSWORD` | Admin password | From opensearch-secret |
| `TLS_HTTP_ENABLED` | TLS enabled flag | Configured from Helm values |
| `DBAAS_ADAPTER_ADDRESS` | DBaaS adapter endpoint | Configured when DBaaS enabled |
| `DBAAS_ADAPTER_USERNAME` | DBaaS adapter username | From dbaas-adapter-secret |
| `DBAAS_ADAPTER_PASSWORD` | DBaaS adapter password | From dbaas-adapter-secret |
| `BACKUP_DAEMON_URL` | Backup daemon URL | Configured from curator service |
| `BACKUP_DAEMON_API_CREDENTIALS_USERNAME` | Backup API username | From curator-secret |
| `BACKUP_DAEMON_API_CREDENTIALS_PASSWORD` | Backup API password | From curator-secret |

**Note**: 
- When `DBAAS_ADAPTER_ADDRESS` is set, the script automatically performs user restoration after migration.
- When `BACKUP_DAEMON_URL` is set (curator enabled), the script automatically creates a backup before migration starts.

## Monitoring Progress

### View Real-Time Logs

```bash
# Watch migration progress
kubectl logs -f $(kubectl get jobs -n opensearch -l component=migration -o jsonpath='{.items[0].metadata.name}')
```

### Check Migration Status

```bash
# View job status
kubectl get jobs -n opensearch -l component=migration

# View detailed job info
kubectl describe job $(kubectl get jobs -n opensearch -l component=migration -o jsonpath='{.items[0].metadata.name}')
```

### Monitor in Grafana

Check the **OpenSearch Indices** dashboard for a panel showing **Indices by OpenSearch Version**.

## Automatic Migration Configuration

Configure the migration job resources and behavior:

```yaml
# values.yaml
migration:
  enabled: false  # Set to true for automatic migration
  
  # Command-line arguments (only used when enabled=true)
  args: []
  # Example:
  #   - "--dry-run"
  #   - "--skip-space-check"
  
  # Resource limits for migration job
  resources:
    requests:
      memory: 256Mi
      cpu: 100m
    limits:
      memory: 1Gi
      cpu: 500m
```

## Troubleshooting

### Insufficient Disk Space

**Error**: "Insufficient disk space for migration"

**Solution**: The migration requires 2x the size of the largest index. Free up space or temporarily skip validation:

```bash
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --skip-space-check
```

### Migration Job Timeout

**Error**: Job exceeds deadline or times out

**Solution**: Increase the job timeout in your values:

```yaml
migration:
  resources:
    limits:
      memory: 2Gi  # Increase memory
      cpu: 1000m   # Increase CPU
```

### Security Reinitialization Failed

**Error**: "Failed to reinitialize security"

**Solution**: Check that the curator ServiceAccount has proper RBAC permissions:

```bash
kubectl auth can-i create pods/exec --as=system:serviceaccount:opensearch:opensearch-curator -n opensearch
```

### Legacy Indices Still Present After Migration

**Error**: Prometheus alert still firing after migration

**Solution**: Wait 10-15 minutes for metrics to refresh, or manually verify:

```bash
curl -k -u $ES_USERNAME:$ES_PASSWORD https://$ES_HOST/_cat/indices?v
```

## Best Practices

1. **Backup is created automatically** if curator is enabled (recommended for all migrations)
2. **Always run dry-run first** to understand what will be migrated
3. **Perform manual migration during maintenance window** for production systems
4. **Verify sufficient disk space** before starting (2x largest index size)
5. **Monitor cluster health** during migration
6. **Test automatic migration** in non-production environments first
7. **Keep the backup ID** from migration logs for recovery if needed

## Support

For issues or questions:
- Check Prometheus alerts for legacy index detection
- Review migration job logs: `kubectl logs -n opensearch <migration-job-pod>`
- Consult the [OpenSearch documentation](https://opensearch.org/docs/latest/upgrade-to/)
