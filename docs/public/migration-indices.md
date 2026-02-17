# Migrating Legacy OpenSearch 1.x Indices

This guide explains how to reindex indices created in OpenSearch 1.x to be compatible with OpenSearch 2.x/3.x.

## Overview

When upgrading an OpenSearch cluster from version 1.x to 2.x or 3.x, indices created in the older version may need to be reindexed to work properly with the new version. The migration script handles this process automatically.

## Migration Approaches

There are two ways to migrate legacy indices:

### 1. Manual Migration (Recommended)
Execute the migration script manually from the curator pod during a planned maintenance window. This gives you full control over timing and monitoring.

### 2. Automatic Migration (Pre-Deploy Hook)
Enable automatic migration to have it run during Helm upgrade:

```yaml
# values.yaml
migration:
  enabled: true  # Automatically migrate during upgrade
```

**Important**: 
- The pre-deploy job **always runs** for OpenSearch 3.x deployments to check for legacy indices
- When `migration.enabled: false` (default): Job **checks only** and will **FAIL the upgrade** if upgrading from 2.x to 3.x with legacy indices present
- When `migration.enabled: true`: Job **performs automatic migration**
- This forces you to either migrate manually first OR explicitly enable automatic migration

### Pre-Deploy Check Behavior

When upgrading to OpenSearch 3.x with `migration.enabled: false` (default):

- ✅ **No legacy indices**: Upgrade proceeds normally
- ✅ **Legacy indices but not 2.x→3.x upgrade**: Warning only, upgrade proceeds
- ❌ **Legacy indices during 2.x→3.x upgrade**: **FAILS** with message:

```
MIGRATION REQUIRED BEFORE UPGRADE TO 3.x

Found X legacy indices created in OpenSearch 1.x
These indices MUST be migrated before upgrading to OpenSearch 3.x

RECOMMENDED STEPS:
1. Install the latest OpenSearch 2.x version first
2. Perform manual migration of legacy indices
3. Verify migration completed successfully
4. Then upgrade to OpenSearch 3.x

ALTERNATIVE: Enable automatic migration in values.yaml:
  migration:
    enabled: true
```

This ensures legacy indices are never accidentally left unmigrated during major upgrades.

## When Migration is Needed

Migration is required when:
- You have indices created in OpenSearch 1.x
- You're running OpenSearch 2.x or 3.x
- Prometheus alert `OpenSearchLegacyIndicesDetected` is firing (see [Alerts](alerts.md#opensearchlegacyindicesdetectedalert))
- Grafana dashboard shows indices with version 1.x

## Running the Migration

The migration script is included in the curator image and can be executed in two ways:

### Option 1: Manual Migration (Recommended)

Manual migration provides better control and is recommended for production environments.

#### Prerequisites

- OpenSearch cluster running version 2.x or 3.x
- Curator pod deployed and running
- Sufficient disk space (2x the size of the largest index)

#### Steps

1. **Identify the curator pod**:
   ```bash
   kubectl get pods -n opensearch -l app=opensearch,component=curator
   ```

2. **Open a terminal in the curator pod**:
   ```bash
   kubectl exec -it <curator-pod-name> -n opensearch -- /bin/bash
   ```

3. **Run the migration script**:
   ```bash
   python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py
   ```

The script uses environment variables that are already configured in the curator pod, so no additional parameters are required.

### Option 2: Automatic Migration via Helm Hook

Enable automatic migration in your Helm values:

```yaml
# values.yaml
migration:
  enabled: true
```

Then upgrade your Helm release:

```bash
helm upgrade opensearch ./opensearch-service -f values.yaml -n opensearch
```

The migration job will run automatically as a pre-upgrade hook before OpenSearch is updated.

**Important Notes**:
- **Pre-deploy check always runs** for OpenSearch 3.x deployments
- When `migration.enabled: false` (default): Checks for legacy indices and **FAILS the upgrade** if upgrading from 2.x→3.x with legacy indices, forcing manual migration first
- When `migration.enabled: true`: Performs automatic migration during upgrade
- Uses curator image and same environment variables
- Runs with hook-weight: 10 (after RBAC setup)
- Job is deleted before each upgrade (before-hook-creation policy)

#### Monitoring Automatic Migration

```bash
# Watch migration job
kubectl get jobs -n opensearch -w

# View migration logs
kubectl logs job/opensearch-migration-1x -n opensearch -f
```

#### Configuring Automatic Migration

```yaml
migration:
  enabled: true
  
  # Optional: dry-run mode
  args:
    - "--dry-run"
  
  # Optional: custom resource limits
  resources:
    requests:
      memory: 512Mi
      cpu: 200m
    limits:
      memory: 1Gi
      cpu: 500m
```

## What the Script Does

The migration script performs the following operations:

### 1. Detection
- Automatically identifies all indices created in OpenSearch 1.x
- Checks index metadata (`version.created` field)
- Skips system indices (starting with `.`)

### 2. Space Validation
- Calculates the size of all indices to migrate
- Identifies the largest index
- Verifies sufficient free space (requires 2x the largest index size)
- Fails with a clear error message if space is insufficient

### 3. Sequential Migration
For each detected 1.x index, the script:
1. **Fetches** mappings and settings from the original index
2. **Reindexes** to a temporary index (`index-name-migration`)
3. **Closes** the original index
4. **Deletes** the original index
5. **Creates** a new index with preserved mappings
6. **Reindexes** from temporary index back to original name
7. **Deletes** the temporary migration index
8. **Opens** the migrated index

### 4. Security and Users
- Backs up OpenSearch security configuration
- Reinitializes security after migration
- Restores DBaaS users if DBaaS adapter is enabled

## Command Line Options

The script supports several command-line arguments:

```bash
# Dry run - show what would be migrated without making changes
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --dry-run

# Skip disk space validation (not recommended)
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --skip-space-check

# Skip security configuration backup
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --skip-security-backup

# Skip DBaaS user restoration
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py --skip-dbaas-restore
```

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

> **Note**: When `DBAAS_ADAPTER_ADDRESS` is set, the script automatically performs user restoration after migration. No additional configuration is needed.

## Monitoring Progress

### View Migration Logs

The script provides detailed logging for each step:

```bash
# Run migration and watch progress
python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py | tee migration.log
```

### Check for Legacy Indices

Before migration:
```bash
# List all indices with version information
curl -k -u $ES_USERNAME:$ES_PASSWORD \
  https://$ES_HOST/_cat/indices?v&h=index,creation.date.string
```

### Monitor in Grafana

Check the **OpenSearch Indices** dashboard:
- Panel: **Indices by OpenSearch Version**
- Shows count of indices created in each OpenSearch version

## Expected Duration

Migration time depends on index size:
- Small index (< 1GB): 1-5 minutes
- Medium index (1-10GB): 5-30 minutes
- Large index (10-100GB): 30-180 minutes
- Very large index (> 100GB): 3+ hours

**Note**: Indices are processed sequentially, one at a time.

## Troubleshooting

### Insufficient Disk Space

**Error**:
```
InsufficientSpaceError: Insufficient disk space for migration!
  Required: 50.00GB
  Available: 30.00GB
```

**Solution**:
1. Free up disk space by deleting old indices
2. Add more storage to OpenSearch data nodes
3. Contact your cluster administrator

### Migration Index Already Exists

**Error**:
```
Migration index 'index-name-migration' already exists
```

**Solution**:
- This is normal if a previous migration was interrupted
- The script will automatically clean up and continue
- No manual intervention needed

### Security Operations Fail

**Error**:
```
Failed to backup security configuration
```

**Solution**:
1. Skip security operations: `--skip-security-backup`
2. Manually back up security config before migration
3. Contact your cluster administrator

## Best Practices

1. **Test First**: Always run with `--dry-run` first to see what will be migrated
2. **Check Space**: Verify sufficient disk space before starting
3. **Maintenance Window**: Run during low-traffic periods
4. **Backup Data**: Take a snapshot before migration
5. **Monitor**: Watch migration progress and cluster health
6. **Verify**: Test application functionality after migration

## Safety and Idempotency

- The migration script is **idempotent** - safe to re-run if it fails
- Temporary migration indices are automatically cleaned up
- Original data is preserved in temporary indices until migration completes
- If migration fails, the script attempts automatic recovery

## Support

If you encounter issues during migration:

1. Check migration logs for specific error messages
2. Review cluster health: `curl https://localhost:9200/_cluster/health`
3. Check available disk space: `df -h`
4. Review this troubleshooting guide
5. Contact your OpenSearch administrator

## See Also

- [Prometheus Alerts](alerts.md#opensearchlegacyindicesdetectedalert) - Alert when legacy indices are detected
- [Installation Guide](installation.md#migrating-legacy-indices) - Initial setup and configuration
- [OpenSearch Documentation](https://opensearch.org/docs/latest/api-reference/document-apis/reindex/) - Reindex API reference
