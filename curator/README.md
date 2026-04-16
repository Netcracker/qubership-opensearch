# What is Qubership OpenSearch Curator?

OpenSearch Curator helps you curate, or manage, your OpenSearch indices and snapshots by:

- Obtaining the full list of indices (or snapshots) from the cluster, as the actionable list
- Iterate through a list of user-defined filters to progressively remove indices (or snapshots) from this actionable list as needed.
- Perform various actions on the items which remain in the actionable list.

OpenSearch Curator does not store OpenSearch backups. It just calls OpenSearch REST API to make it to start backup process.
And all backups are stored on OpenSearch side, OpenSearch Curator only stores meta information about backups (timestamp, size, list of indices, etc.).

# API Usage

For POST operations you must specify user/pass from `BACKUP_DAEMON_API_CREDENTIALS_USERNAME` and `BACKUP_DAEMON_API_CREDENTIALS_PASSWORD` env parameters so that you can use REST api to run backup tasks:

## Backup

### Full Manual Backup

If you want to make a backup of all OpenSearch indices data, you need to run the following command:

```
curl -XPOST -u username:password http://localhost:8080/backup
```

After executing the command you receive name of folder where the backup is stored. For example,
`20190321T080000`.

### Granular Backup

If you want the backup to be performed for specified database prefixes you can specify them in parameter `dbs`. For example:

```
curl -XPOST -u username:password -v -H "Content-Type: application/json" -d '{"dbs":["db1","db2"]}'  http://localhost:8080/backup
```

### Not Evictable Backup

If backup should not be evicted automatically, it is necessary to add `allow_eviction` property
with value `False` to the request body. For example,

```
curl -XPOST -u username:password -v -H "Content-Type: application/json" -d '{"allow_eviction":"False"}' http://localhost:8080/backup
```

### Backup Eviction

#### Evict Backup by ID

If you want to remove specific backup, you should run the following command:

```
curl -XPOST -u username:password http://localhost:8080/evict/<backup_id>
```

where `backup_id` is the name of necessary backup, e.g. `20190321T080000`. If operation is
successful, you see the following text: `Backup <backup_id> successfully removed`.

### Backup Status

If backup is in progress, you can check its status running the following command:

```
curl -XGET http://localhost:8080/jobstatus/<backup_id>
```

where `backup_id` is backup name received at the backup execution step. The result is JSON with
the following information:

* `status` is status of operation, possible options: Successful, Queued, Processing, Failed
* `message` is description of error (optional field)
* `vault` is name of vault used in recovery
* `type` is type of operation, possible options: backup, restore
* `err` is last 5 lines of error logs if `status = Failed`, None otherwise
* `task_id` is identifier of the task

### Backup Information

To get the backup information, use the following command:

```
curl -XGET http://localhost:8080/listbackups/<backup_id>
```

where `backup_id` is the name of necessary backup. The command returns JSON string with data about
particular backup:

* `ts` is UNIX timestamp of backup
* `spent_time` is time spent on backup (in ms)
* `db_list` is list of stored database prefixes
* `id` is backup name
* `size` is size of backup (in bytes)
* `evictable` is _true_ if backup is evictable, _false_ otherwise
* `locked` is _true_ if backup is locked (either process isn't finished, or it failed somehow)
* `exit_code` is exit code of backup script
* `failed` is _true_ if backup failed, _false_ otherwise
* `valid` is _true_ if backup is valid, _false_ otherwise

## Recovery

To recover data from certain backup, you need to specify JSON with information about a backup name (`vault`).

If you need to restore only specific databases, use `dbs` parameter in JSON. **Pay attention**, that you can use this parameter only to restore from `granular` snapshots.

```
curl -XPOST -u username:password -v -H "Content-Type: application/json" -d '{"vault":"20190321T080000", "dbs":["index1","index2"]}' http://localhost:8080/restore
```

If you want to rename database entities during recovery, you need to specify `changeDbName` parameter in JSON. Like `dbs`, you can use `changeDbName` parameter only to restore from `granular` snapshots. For example,

```
curl -XPOST -u username:password -v -H "Content-Type: application/json" -d  '{"vault":"20190321T080000", "dbs":["db1","db2","db3"], "changeDbNames":{"db1":"new_db1_name","db2":"new_db2_name"}}' http://localhost:8080/restore
```

This functionality will leave `db1` and `db2` databases as is and restore `db1` and `db2` into new (or existing) databases called `new_db1_name` and `new_db2_name`. The `db3` database will be rewritten, because it's not in `changeDbNames` list.

If you need to clean resources before recovery, use `clean` parameter in JSON. It removes indices, aliases, index (old and new) and component templates from OpenSearch depending on recovery type. For `full` recovery it removes all resources. For `granular` recovery without renaming - only resources prefixed with the specified databases. For `granular` recovery with renaming - only resources prefixed with the specified databases or renaming pattern in `changeDbNames` list. For example,

```
curl -XPOST -u username:password -v -H "Content-Type: application/json" -d  '{"vault":"20190321T080000", "dbs":["db1","db2","db3"], "changeDbNames":{"db1":"new_db1_name","db2":"new_db2_name"}, "clean":"true"}' http://localhost:8080/restore
```

This functionality will remove indices, templates and aliases of databases that should be restored. For example, it deletes resources by `new_db1_name`, `new_db2_name` and `db3` prefixes as `db1` and `db2` databases are to be renamed and `db3` is to be restored with the same name.

**Note**: You can use `clean` parameter if you need data for all or corresponding databases only from backup, or common recovery fails due to a configuration conflict between existing and restored resources.

### Users Recovery

**Attention**: Curator doesn't restore users that are created using OpenSearch API. Curator restores only DBaaS users. 

To recover users via DBaaS during restore procedure, you need to specify JSON with specific value `skip_users_recovery`.

It works for both granular and full backup. By default, users recovery is enabled if DBaaS parameters specified correctly.

```
curl -XPOST -u username:password -v -H "Content-Type: application/json" -d '{"vault":"20190321T080000", "skip_users_recovery":"true"}' http://localhost:8080/restore
```

As a response you receive `task_id`, which can be used to check _Recovery Status_.

### Recovery Status

If recovery is in progress, you can check its status running the following command:

```
curl -XGET http://localhost:8080/jobstatus/<task_id>
```

where `task_id` is task id received at the recovery execution step.

## Backups List

To receive list of collected backups you need to use the following command:

```
curl -XGET http://localhost:8080/listbackups
```

It returns JSON with list of backup names.

## Backup Daemon Health

If you want to know the state of Backup Daemon, you should use the following command:

```
curl -XGET http://localhost:8080/health
```

As a result you receive JSON with information:

```
"status": status of backup daemon   
"backup_queue_size": backup daemon queue size (if > 0 then there are 1 or tasks waiting for execution)
 "storage": storage info:
  "total_space": total storage space in bytes
  "dump_count": number of backup
  "free_space": free space left in bytes
  "size": used space in bytes
  "total_inodes": total number of inodes on storage
  "free_inodes": free number of inodes on storage
  "used_inodes": used number of inodes on storage
  "last": last backup metrics
    "metrics['exit_code']": exit code of script 
    "metrics['exception']": python exception if backup failed
    "metrics['spent_time']": spent time
    "metrics['size']": backup size in bytes
    "failed": is failed or not
    "locked": is locked or not
    "id": vault name of backup
    "ts": timestamp of backup  
  "lastSuccessful": last succesfull backup metrics
    "metrics['exit_code']": exit code of script 
    "metrics['spent_time']": spent time
    "metrics['size']": backup size in bytes
    "failed": is failed or not
    "locked": is locked or not
    "id": vault name of backup
    "ts": timestamp of backup
```

# Scheduled snapshots cleanup

Curator is able to perform scheduled snapshots cleanup. However, it is not possible to provide common template that can be easily configured for different use cases. 
So actions configuration for cleanup should be provided. It is highly recommended to include cleanup actions as part of snapshot creation action file (snapshot.yml) that is mounted to /opt/OpenSearch-curator/actions/ in container.

For more information on different filter types (such as [age](https://www.elastic.co/guide/en/elasticsearch/client/curator/current/filtertype_age.html) and [period](https://www.elastic.co/guide/en/elasticsearch/client/curator/current/filtertype_period.html)) please check [curator documentation](https://www.elastic.co/guide/en/elasticsearch/client/curator/5.4/index.html).

## Example of configuration
Here is sample of configuration (for development) that will create snapshots every 10 minutes. After 1 hour passes it will keep only hourly snapshots.

`BACKUP_SCHEDULE` environmental variable is set to "0,10,20,30,40,50 * * * *"
period filter is used to keep only one hourly snapshot after 1 hour passes since snapshot creation:

```
actions:
  1:
    action: snapshot
    description: >-
      Snapshot selected indices to 'repository' with the snapshot name or name
      pattern in 'name'.  Use all other options as assigned
    options:
      repository: ${SNAPSHOT_REPOSITORY_NAME}
      ignore_unavailable: False
      include_global_state: False
      partial: False
      wait_for_completion: True
      skip_repo_fs_check: False
      timeout_override:
      continue_if_exception: False
      disable_action: False
    filters:
    - filtertype: opened
      exclude: false
  2:
    action: delete_snapshots
    description: >-
      Cleanup old snapshots in 'repository' based on theirs age
    options:
      repository: ${SNAPSHOT_REPOSITORY_NAME}
      retry_interval: 15
      retry_count: 3
      disable_action: False
    filters:
    - filtertype: period  
      source: creation_date
      range_from: -1
      range_to: -1
      unit: hours
    - filtertype: count
      count: 1
      source: creation_date
      use_age: True
      reverse: False
      exclude: True
```

## Eviction Policy

OpenSearch Curator provides eviction policy to remove obsolete snapshots. You can change eviction policy via environment variable `EVICTION_POLICY`.

Eviction policy is a comma-separated string of policies written as `$start_time/$interval`. This policy splits all backups older than $start_time to numerous time intervals $interval time long. Then it deletes all backups in every interval except the newest one.

For example:
* `1d/7d` policy means "take all backups older than one day, split them in groups by 7-days interval, and leave only the newest"
* `0/1h` means "take all backups older than now, split them in groups by 1 hour and leave only the newest"