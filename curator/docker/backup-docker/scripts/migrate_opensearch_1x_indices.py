#!/usr/bin/env python3
# Copyright 2024-2025 NetCracker Technology Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
OpenSearch 1.x Index Migration Script

This script migrates indices created in OpenSearch 1.x to be compatible with OpenSearch 2.x/3.x.
It performs reindexing to update the index metadata and mappings.

Features:
- Detects indices created in OpenSearch 1.x
- Validates available disk space before migration
- Sequential processing with detailed logging
- Security configuration backup and restoration
- DBaaS user restoration support
- Idempotent operations (safe to re-run)
"""

import argparse
import base64
import json
import logging
import os
import sys
import time
from typing import Dict, List, Optional, Tuple
import subprocess
import requests
from requests.auth import HTTPBasicAuth


# Configuration from environment variables
# Using curator's environment variable names for consistency
ES_HOST = os.environ.get('ES_HOST', 'opensearch-internal:9200')
ES_USERNAME = os.environ.get('ES_USERNAME', 'admin')
ES_PASSWORD = os.environ.get('ES_PASSWORD', 'admin')
TLS_HTTP_ENABLED = os.environ.get('TLS_HTTP_ENABLED', 'false').lower() == 'true'

# Build OpenSearch endpoint from curator variables
if TLS_HTTP_ENABLED:
    OPENSEARCH_ENDPOINT = f"https://{ES_HOST}" if not ES_HOST.startswith('http') else ES_HOST
else:
    OPENSEARCH_ENDPOINT = f"http://{ES_HOST}" if not ES_HOST.startswith('http') else ES_HOST

# DBaaS integration (from curator variables)
DBAAS_ADAPTER_ADDRESS = os.environ.get('DBAAS_ADAPTER_ADDRESS', '')
DBAAS_ADAPTER_USERNAME = os.environ.get('DBAAS_ADAPTER_USERNAME', '')
DBAAS_ADAPTER_PASSWORD = os.environ.get('DBAAS_ADAPTER_PASSWORD', '')
# Note: DBAAS_ADAPTER_ADDRESS is only set when DBaaS is enabled in curator

# Backup daemon integration (from curator variables)
# Default to localhost when running from curator pod
BACKUP_DAEMON_URL = os.environ.get('BACKUP_DAEMON_URL', 'http://localhost:8080')
BACKUP_DAEMON_API_CREDENTIALS_USERNAME = os.environ.get('BACKUP_DAEMON_API_CREDENTIALS_USERNAME', '')
BACKUP_DAEMON_API_CREDENTIALS_PASSWORD = os.environ.get('BACKUP_DAEMON_API_CREDENTIALS_PASSWORD', '')

# Pre-deploy job configuration
MIGRATION_MODE = os.environ.get('MIGRATION_MODE', 'manual')  # 'manual', 'migration', or 'pre-deploy-check'
OPENSEARCH_NAMESPACE = os.environ.get('OPENSEARCH_NAMESPACE', os.environ.get('NAMESPACE'))
# For security reinit via plugins.security.disabled (does not require TLS / securityadmin.sh)
OPENSEARCH_CONFIG_SECRET_NAME = os.environ.get('OPENSEARCH_CONFIG_SECRET_NAME', '')
OPENSEARCH_STATEFULSET_NAMES = os.environ.get('OPENSEARCH_STATEFULSET_NAMES', '')   # comma-separated
OPENSEARCH_DEPLOYMENT_NAMES = os.environ.get('OPENSEARCH_DEPLOYMENT_NAMES', '')     # comma-separated

# Constants
MIGRATION_SUFFIX = '-migration'
USERS_RECOVERY_DONE_STATE = 'done'
USERS_RECOVERY_FAILED_STATE = 'failed'
USERS_RECOVERY_RUNNING_STATE = 'running'
USERS_RECOVERY_IDLE_STATE = 'idle'
DBAAS_TIMEOUT = 240
DBAAS_INTERVAL = 10
CLUSTER_READY_TIMEOUT = 600   # seconds to wait for cluster green + API
CLUSTER_READY_INTERVAL = 15   # poll interval
SECURITY_DISABLED_LINE = 'plugins.security.disabled: true'
OPENDISTRO_SECURITY_INDEX = '.opendistro_security'

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='[%(asctime)s][%(levelname)s] %(message)s',
    datefmt='%Y-%m-%dT%H:%M:%S'
)
logger = logging.getLogger(__name__)


class MigrationError(Exception):
    """Base exception for migration errors"""
    pass


class InsufficientSpaceError(MigrationError):
    """Raised when there is insufficient disk space for migration"""
    pass


class OpenSearchClient:
    """OpenSearch HTTP client with authentication"""
    
    def __init__(self, endpoint: str, username: str, password: str):
        self.endpoint = endpoint.rstrip('/')
        self.auth = HTTPBasicAuth(username, password)
        self.session = requests.Session()
        self.session.auth = self.auth
        # Disable SSL verification warnings for self-signed certs
        self.session.verify = False
        requests.packages.urllib3.disable_warnings()
    
    def request(self, method: str, path: str, **kwargs) -> requests.Response:
        """Make an HTTP request to OpenSearch"""
        url = f"{self.endpoint}/{path.lstrip('/')}"
        try:
            response = self.session.request(method, url, **kwargs)
            response.raise_for_status()
            return response
        except requests.exceptions.RequestException as e:
            logger.error(f"Request failed: {method} {url} - {str(e)}")
            raise MigrationError(f"OpenSearch request failed: {str(e)}")
    
    def get(self, path: str, **kwargs) -> requests.Response:
        return self.request('GET', path, **kwargs)
    
    def post(self, path: str, **kwargs) -> requests.Response:
        return self.request('POST', path, **kwargs)
    
    def put(self, path: str, **kwargs) -> requests.Response:
        return self.request('PUT', path, **kwargs)
    
    def delete(self, path: str, **kwargs) -> requests.Response:
        return self.request('DELETE', path, **kwargs)


class IndexMigrator:
    """Handles migration of OpenSearch 1.x indices"""
    
    def __init__(self, client: OpenSearchClient):
        self.client = client
    
    def get_all_indices(self) -> List[Dict]:
        """Get all indices in the cluster"""
        logger.info("Fetching all indices...")
        response = self.client.get('_cat/indices?format=json&h=index,pri.store.size,store.size,creation.date.string')
        return response.json()
    
    def get_index_settings(self, index: str) -> Dict:
        """Get settings for a specific index"""
        response = self.client.get(f"{index}/_settings")
        return response.json()
    
    def get_index_mappings(self, index: str) -> Dict:
        """Get mappings for a specific index"""
        response = self.client.get(f"{index}/_mappings")
        return response.json()
    
    def is_1x_index(self, index: str) -> bool:
        """
        Check if an index was created in OpenSearch 1.x.
        
        Uses XOR decoding approach from indices_version_metric.py:
        - version.created is XOR'd with 0x08000000 to get the real version
        - After decoding and removing suffix, format is XXYYZZ where XX is major version
        - Version < 20000 means OpenSearch 1.x
        """
        try:
            settings = self.get_index_settings(index)
            index_settings = settings.get(index, {}).get('settings', {}).get('index', {})
            
            # Check version in settings
            version_created = index_settings.get('version', {}).get('created', '')
            if version_created:
                version_created_num = int(version_created)
                
                # XOR with 0x08000000 to decode the version
                decoded = version_created_num ^ 0x08000000
                
                # Remove last 2 digits (99 suffix)
                version_num = decoded // 100
                
                # Check if major version is 1.x (version_num < 20000)
                if version_num < 20000:
                    # Extract readable version for logging
                    major = version_num // 10000
                    minor = (version_num // 100) % 100
                    patch = version_num % 100
                    version_str = f"{major}.{minor}.{patch}"
                    logger.info(f"Index '{index}' was created in OpenSearch 1.x (version: {version_str}, encoded: {version_created})")
                    return True
            
            return False
        except Exception as e:
            logger.warning(f"Could not determine version for index '{index}': {str(e)}")
            return False
    
    def get_1x_indices(self) -> List[str]:
        """Get all indices created in OpenSearch 1.x, including system indices except .opendistro_security"""
        logger.info("Detecting indices created in OpenSearch 1.x...")
        all_indices = self.get_all_indices()
        
        indices_1x = []
        security_needs_reinit = False
        
        for idx_info in all_indices:
            index = idx_info.get('index', '')
            
            # Skip migration indices
            if index.endswith(MIGRATION_SUFFIX):
                continue
            
            # Special handling for .opendistro_security
            if index == '.opendistro_security':
                if self.is_1x_index(index):
                    logger.warning(f"Security index '{index}' was created in OpenSearch 1.x (version check)")
                    logger.warning(f"Security will be reinitialized after migration")
                    security_needs_reinit = True
                # Never migrate .opendistro_security - always reinit instead
                continue
            
            # Include all other indices (including system indices starting with .)
            if self.is_1x_index(index):
                indices_1x.append(index)
        
        logger.info(f"Found {len(indices_1x)} indices created in OpenSearch 1.x: {indices_1x}")
        if security_needs_reinit:
            logger.info(f"Security index needs reinitialization (will be performed after migration)")
        
        return indices_1x, security_needs_reinit
    
    def get_cluster_stats(self) -> Dict:
        """Get cluster statistics including disk usage"""
        response = self.client.get('_cluster/stats')
        return response.json()
    
    def get_nodes_stats(self) -> Dict:
        """Get node statistics including disk space"""
        response = self.client.get('_nodes/stats/fs')
        return response.json()
    
    def get_index_size_bytes(self, index: str) -> int:
        """Get index size in bytes"""
        try:
            response = self.client.get(f'_cat/indices/{index}?format=json&h=pri.store.size,store.size&bytes=b')
            data = response.json()
            if data:
                # Use primary store size for more accurate calculation
                size_str = data[0].get('pri.store.size', '0')
                return int(size_str)
            return 0
        except Exception as e:
            logger.warning(f"Could not get size for index '{index}': {str(e)}")
            return 0
    
    def get_available_disk_space(self) -> int:
        """Get available disk space in bytes"""
        try:
            nodes_stats = self.get_nodes_stats()
            min_available = None
            
            for node_id, node_data in nodes_stats.get('nodes', {}).items():
                fs_data = node_data.get('fs', {}).get('total', {})
                available = fs_data.get('available_in_bytes', 0)
                
                if min_available is None or available < min_available:
                    min_available = available
            
            logger.info(f"Minimum available disk space across nodes: {self._bytes_to_human(min_available)}")
            return min_available or 0
        except Exception as e:
            logger.error(f"Could not determine available disk space: {str(e)}")
            raise MigrationError("Failed to determine available disk space")
    
    def _bytes_to_human(self, bytes_val: int) -> str:
        """Convert bytes to human-readable format"""
        for unit in ['B', 'KB', 'MB', 'GB', 'TB']:
            if bytes_val < 1024.0:
                return f"{bytes_val:.2f}{unit}"
            bytes_val /= 1024.0
        return f"{bytes_val:.2f}PB"
    
    def validate_disk_space(self, indices: List[str]) -> Tuple[str, int]:
        """
        Validate sufficient disk space for migration
        Returns: (largest_index, largest_size)
        """
        logger.info("Validating available disk space...")
        
        if not indices:
            logger.info("No indices to migrate, skipping disk space validation")
            return None, 0
        
        # Get size of each index
        index_sizes = {}
        for index in indices:
            size = self.get_index_size_bytes(index)
            index_sizes[index] = size
            logger.info(f"Index '{index}' size: {self._bytes_to_human(size)}")
        
        # Find largest index
        largest_index = max(index_sizes.items(), key=lambda x: x[1])
        largest_name, largest_size = largest_index
        required_space = largest_size * 2
        
        logger.info(f"Largest index: '{largest_name}' ({self._bytes_to_human(largest_size)})")
        logger.info(f"Required free space: {self._bytes_to_human(required_space)} (2x largest index)")
        
        # Get available disk space
        available_space = self.get_available_disk_space()
        
        if available_space < required_space:
            error_msg = (
                f"Insufficient disk space for migration!\n"
                f"  Required: {self._bytes_to_human(required_space)}\n"
                f"  Available: {self._bytes_to_human(available_space)}\n"
                f"  Largest index: '{largest_name}' ({self._bytes_to_human(largest_size)})"
            )
            logger.error(error_msg)
            raise InsufficientSpaceError(error_msg)
        
        logger.info(f"✓ Sufficient disk space available: {self._bytes_to_human(available_space)}")
        return largest_name, largest_size
    
    def close_index(self, index: str):
        """Close an index"""
        logger.info(f"Closing index '{index}'...")
        try:
            self.client.post(f"{index}/_close")
            logger.info(f"✓ Index '{index}' closed successfully")
        except Exception as e:
            raise MigrationError(f"Failed to close index '{index}': {str(e)}")
    
    def open_index(self, index: str):
        """Open an index"""
        logger.info(f"Opening index '{index}'...")
        try:
            self.client.post(f"{index}/_open")
            logger.info(f"✓ Index '{index}' opened successfully")
        except Exception as e:
            raise MigrationError(f"Failed to open index '{index}': {str(e)}")
    
    def delete_index(self, index: str):
        """Delete an index"""
        logger.info(f"Deleting index '{index}'...")
        try:
            self.client.delete(index)
            logger.info(f"✓ Index '{index}' deleted successfully")
        except Exception as e:
            raise MigrationError(f"Failed to delete index '{index}': {str(e)}")
    
    def index_exists(self, index: str) -> bool:
        """Check if an index exists"""
        try:
            self.client.get(f"{index}")
            return True
        except:
            return False
    
    def create_index_with_mappings(self, index: str, mappings: Dict, settings: Dict = None):
        """Create an index with specific mappings and settings"""
        logger.info(f"Creating index '{index}' with mappings...")
        
        body = {}
        if settings:
            body['settings'] = settings
        if mappings:
            body['mappings'] = mappings
        
        try:
            self.client.put(index, json=body, headers={'Content-Type': 'application/json'})
            logger.info(f"✓ Index '{index}' created successfully")
        except Exception as e:
            raise MigrationError(f"Failed to create index '{index}': {str(e)}")
    
    def reindex(self, source: str, dest: str, wait_for_completion: bool = True):
        """Reindex from source to destination"""
        logger.info(f"Reindexing from '{source}' to '{dest}'...")
        
        body = {
            "source": {"index": source},
            "dest": {"index": dest}
        }
        
        try:
            params = {}
            if wait_for_completion:
                params['wait_for_completion'] = 'true'
            
            response = self.client.post('_reindex', json=body, params=params,
                                       headers={'Content-Type': 'application/json'},
                                       timeout=3600)  # 1 hour timeout
            
            result = response.json()
            
            if wait_for_completion:
                total = result.get('total', 0)
                created = result.get('created', 0)
                failures = result.get('failures', [])
                
                logger.info(f"Reindexing completed: {created}/{total} documents transferred")
                
                if failures:
                    logger.warning(f"Reindexing had {len(failures)} failures")
                    for failure in failures[:5]:  # Log first 5 failures
                        logger.warning(f"  Failure: {json.dumps(failure)}")
                
                if created == 0 and total > 0:
                    raise MigrationError(f"Reindexing failed: no documents were created")
            else:
                task_id = result.get('task')
                logger.info(f"Reindexing task started: {task_id}")
            
        except Exception as e:
            raise MigrationError(f"Failed to reindex from '{source}' to '{dest}': {str(e)}")
    
    def migrate_index(self, index: str, is_system_index: bool = False):
        """
        Migrate a single index from 1.x to 2.x/3.x format
        
        For system indices: If migration fails, delete the index (it will be recreated)
        For user indices: Fail immediately on error with cleanup
        
        Correct Process:
        1. Get mappings and settings from original index
        2. Create migration index with those settings/mappings
        3. Reindex original to migration index
        4. Delete original index
        5. Create original index with migration settings/mappings
        6. Reindex migration back to original
        7. Delete migration index
        
        IMPORTANT: If any step fails, stop and restore state. Never delete data without successful reindex.
        """
        logger.info(f"=" * 80)
        logger.info(f"Starting migration for index: '{index}'")
        if is_system_index:
            logger.info(f"Note: This is a system index (will be deleted if migration fails)")
        logger.info(f"=" * 80)
        
        migration_index = f"{index}{MIGRATION_SUFFIX}"
        original_deleted = False  # Track if we deleted original index
        
        try:
            # Step 1: Get mappings and settings from original index
            logger.info(f"[1/7] Fetching mappings and settings from '{index}'...")
            mappings_response = self.get_index_mappings(index)
            mappings = mappings_response.get(index, {}).get('mappings', {})
            
            settings_response = self.get_index_settings(index)
            index_settings = settings_response.get(index, {}).get('settings', {}).get('index', {})
            
            # Extract relevant settings (exclude version-specific ones)
            preserved_settings = {}
            for key in ['number_of_shards', 'number_of_replicas', 'refresh_interval']:
                if key in index_settings:
                    preserved_settings[key] = index_settings[key]
            
            logger.info(f"      Retrieved mappings and settings successfully")
            logger.info(f"      Shards: {preserved_settings.get('number_of_shards', 'default')}, "
                       f"Replicas: {preserved_settings.get('number_of_replicas', 'default')}")
            
            # Step 2: Create migration index with original settings and mappings
            logger.info(f"[2/7] Creating migration index '{migration_index}' with original settings/mappings...")
            
            # Check if migration index already exists (from previous failed run)
            if self.index_exists(migration_index):
                logger.warning(f"      Migration index '{migration_index}' already exists - cleaning up from previous run")
                self.delete_index(migration_index)
            
            # Create migration index with clean mappings
            clean_mappings = self._clean_mappings(mappings) if mappings else {}
            if clean_mappings or preserved_settings:
                self.create_index_with_mappings(
                    migration_index, 
                    clean_mappings,
                    {'index': preserved_settings} if preserved_settings else None
                )
            else:
                # Create empty index (will be created during reindex)
                logger.info(f"      No mappings or settings to preserve, migration index will be auto-created")
            
            logger.info(f"      Migration index '{migration_index}' created successfully")
            
            # Step 3: Reindex original to migration index
            logger.info(f"[3/7] Reindexing data from '{index}' to '{migration_index}'...")
            self.reindex(index, migration_index)
            logger.info(f"      Data successfully copied to migration index")
            
            # Step 4: Delete original index (ONLY after successful reindex)
            logger.info(f"[4/7] Deleting original index '{index}'...")
            self.delete_index(index)
            original_deleted = True
            logger.info(f"      Original index deleted (data preserved in migration index)")
            
            # Step 5: Create original index with migrated settings/mappings
            logger.info(f"[5/7] Creating new '{index}' with migrated format...")
            if clean_mappings or preserved_settings:
                self.create_index_with_mappings(
                    index,
                    clean_mappings,
                    {'index': preserved_settings} if preserved_settings else None
                )
            else:
                logger.info(f"      Index will be auto-created during reindex")
            logger.info(f"      New index '{index}' created successfully")
            
            # Step 6: Reindex migration back to original
            logger.info(f"[6/7] Reindexing data from '{migration_index}' back to '{index}'...")
            self.reindex(migration_index, index)
            logger.info(f"      Data successfully restored to original index name")
            
            # Step 7: Delete migration index (cleanup)
            logger.info(f"[7/7] Deleting temporary migration index '{migration_index}'...")
            self.delete_index(migration_index)
            logger.info(f"      Migration index cleaned up")
            
            logger.info(f"✓ SUCCESS: Index '{index}' migrated successfully")
            logger.info(f"=" * 80)
            
        except Exception as e:
            logger.error(f"✗ FAILED: Migration failed for index '{index}': {str(e)}")
            logger.error(f"=" * 80)
            
            if is_system_index:
                # For system indices, delete and let OpenSearch recreate them
                logger.warning(f"System index '{index}' migration failed - will delete it")
                logger.warning(f"OpenSearch or plugins will recreate this index automatically")
                try:
                    self._cleanup_and_delete_system_index(index, migration_index, original_deleted)
                    logger.info(f"✓ System index '{index}' deleted (will be recreated)")
                    logger.info(f"=" * 80)
                except Exception as cleanup_error:
                    logger.error(f"Failed to cleanup system index '{index}': {str(cleanup_error)}")
                    raise
            else:
                # For user indices, attempt restore and fail
                logger.error(f"Attempting to restore index '{index}' from backup...")
                self._cleanup_failed_migration(index, migration_index, original_deleted)
                raise
    
    def _clean_mappings(self, mappings: Dict) -> Dict:
        """Remove version-specific fields from mappings"""
        # Deep copy to avoid modifying original
        import copy
        clean = copy.deepcopy(mappings)
        
        # Remove _meta fields that might cause issues
        if '_meta' in clean:
            meta = clean['_meta']
            # Keep user-defined metadata, remove system metadata
            system_fields = ['version', 'created', 'created_by']
            for field in system_fields:
                meta.pop(field, None)
        
        return clean
    
    def _cleanup_failed_migration(self, original: str, migration: str, original_deleted: bool):
        """
        Attempt to cleanup and restore after a failed migration.
        
        Args:
            original: Original index name
            migration: Migration index name
            original_deleted: True if original index was already deleted
        """
        logger.info("Attempting cleanup and restore after failed migration...")
        
        try:
            # If original was deleted and migration exists, restore from migration
            if original_deleted and self.index_exists(migration):
                logger.info(f"Original index '{original}' was deleted - restoring from migration index")
                
                # Reindex from migration back to original
                logger.info(f"Restoring data from '{migration}' to '{original}'...")
                self.reindex(migration, original)
                
                # Delete migration index
                logger.info(f"Cleaning up migration index '{migration}'...")
                self.delete_index(migration)
                
                logger.info(f"✓ Successfully restored original index '{original}' from migration backup")
                
            elif self.index_exists(migration):
                # Original still exists, just cleanup migration index
                logger.info(f"Original index '{original}' still exists - just cleaning up migration index")
                self.delete_index(migration)
                logger.info(f"✓ Migration index '{migration}' cleaned up")
                
            elif not self.index_exists(original):
                # Both indices are gone - data loss!
                logger.error(f"✗ CRITICAL: Both original and migration indices are missing!")
                logger.error(f"Data for index '{original}' may be lost - restore from backup required")
                
            else:
                # Original exists, migration doesn't - normal state
                logger.info(f"Original index '{original}' exists, no cleanup needed")
                
        except Exception as e:
            logger.error(f"✗ Cleanup failed: {str(e)}")
            logger.error("Manual intervention may be required!")
            if original_deleted:
                logger.error(f"CRITICAL: Original index '{original}' was deleted during migration")
                logger.error(f"Check if migration index '{migration}' exists and restore manually")
    
    def _cleanup_and_delete_system_index(self, original: str, migration: str, original_deleted: bool):
        """
        Delete a system index that failed to migrate (it will be recreated automatically).
        
        Args:
            original: Original index name
            migration: Migration index name  
            original_deleted: True if original index was already deleted
        """
        logger.info(f"Cleaning up failed system index migration for '{original}'...")
        
        try:
            # Delete migration index if it exists
            if self.index_exists(migration):
                logger.info(f"Deleting migration index '{migration}'...")
                self.delete_index(migration)
            
            # Delete original index if it still exists
            if not original_deleted and self.index_exists(original):
                logger.info(f"Deleting original index '{original}'...")
                # Try to open it first in case it's closed
                try:
                    self.open_index(original)
                except:
                    pass
                self.delete_index(original)
            
            logger.info(f"✓ System index '{original}' deleted successfully")
            
        except Exception as e:
            logger.error(f"✗ Failed to delete system index '{original}': {str(e)}")
            raise


class SecurityReinitViaDisable:
    """
    Reinitialize the OpenSearch security plugin without securityadmin.sh / TLS.

    Full flow
    ---------
    1. Append ``plugins.security.disabled: true`` to opensearch.yml in the
       opensearch-config Kubernetes Secret.
    2. Rolling-restart every OpenSearch StatefulSet and Deployment so all pods
       pick up the new config (security plugin is now bypassed).
    3. Wait until the cluster reports green status and the REST API is reachable
       (no authentication required while security is disabled).
    4. DELETE the ``.opendistro_security`` index so it will be freshly created
       on the next boot.
    5. Remove ``plugins.security.disabled: true`` from the secret.
    6. Rolling-restart all pods again so they come up with security enabled.
    7. Wait until the cluster is green and the REST API is reachable *with*
       authentication (security plugin initialised a fresh security index).
    """

    def __init__(self, client: OpenSearchClient):
        self.client = client

    # ------------------------------------------------------------------
    # Low-level kubectl helpers
    # ------------------------------------------------------------------

    def _kubectl(self, args: List[str], check: bool = True) -> subprocess.CompletedProcess:
        ns = OPENSEARCH_NAMESPACE or 'default'
        cmd = ['kubectl', '-n', ns] + args
        return subprocess.run(cmd, capture_output=True, text=True, check=check)

    # ------------------------------------------------------------------
    # Secret helpers
    # ------------------------------------------------------------------

    def _get_opensearch_yml(self) -> Optional[str]:
        """Read and base64-decode the opensearch.yml key from the config secret."""
        if not OPENSEARCH_CONFIG_SECRET_NAME or not OPENSEARCH_NAMESPACE:
            logger.error("OPENSEARCH_CONFIG_SECRET_NAME / OPENSEARCH_NAMESPACE must be set")
            return None
        try:
            result = self._kubectl([
                'get', 'secret', OPENSEARCH_CONFIG_SECRET_NAME,
                '-o', r'jsonpath={.data.opensearch\.yml}'
            ])
            if result.returncode != 0 or not result.stdout.strip():
                logger.error(f"Cannot read opensearch.yml from secret: {result.stderr}")
                return None
            return base64.b64decode(result.stdout.strip()).decode('utf-8', errors='replace')
        except Exception as e:
            logger.error(f"Failed to read opensearch.yml from secret: {e}")
            return None

    def _patch_opensearch_yml(self, content: str) -> bool:
        """Base64-encode *content* and write it back to the config secret."""
        if not OPENSEARCH_CONFIG_SECRET_NAME or not OPENSEARCH_NAMESPACE:
            return False
        try:
            b64 = base64.b64encode(content.encode('utf-8')).decode('ascii')
            patch = json.dumps({'data': {'opensearch.yml': b64}})
            result = self._kubectl([
                'patch', 'secret', OPENSEARCH_CONFIG_SECRET_NAME,
                '--type', 'merge', '-p', patch,
            ])
            if result.returncode != 0:
                logger.error(f"Failed to patch secret: {result.stderr}")
                return False
            return True
        except Exception as e:
            logger.error(f"Failed to patch secret: {e}")
            return False

    def _add_security_disabled(self) -> bool:
        """Append ``plugins.security.disabled: true`` to opensearch.yml in the secret."""
        content = self._get_opensearch_yml()
        if content is None:
            return False
        if SECURITY_DISABLED_LINE in content:
            logger.info("plugins.security.disabled: true already present — nothing to add")
            return True
        new_content = content.rstrip('\n') + '\n' + SECURITY_DISABLED_LINE + '\n'
        logger.info(f"Adding '{SECURITY_DISABLED_LINE}' to opensearch.yml in secret "
                    f"'{OPENSEARCH_CONFIG_SECRET_NAME}'")
        return self._patch_opensearch_yml(new_content)

    def _remove_security_disabled(self) -> bool:
        """Remove ``plugins.security.disabled: true`` from opensearch.yml in the secret."""
        content = self._get_opensearch_yml()
        if content is None:
            return False
        lines = [ln for ln in content.splitlines() if ln.strip() != SECURITY_DISABLED_LINE]
        new_content = '\n'.join(lines)
        if new_content and not new_content.endswith('\n'):
            new_content += '\n'
        logger.info(f"Removing '{SECURITY_DISABLED_LINE}' from opensearch.yml in secret "
                    f"'{OPENSEARCH_CONFIG_SECRET_NAME}'")
        return self._patch_opensearch_yml(new_content)

    # ------------------------------------------------------------------
    # Pod restart helpers
    # ------------------------------------------------------------------

    def _restart_opensearch_workloads(self) -> bool:
        """Issue ``kubectl rollout restart`` for every configured workload."""
        ok = True
        sts_names = [n.strip() for n in OPENSEARCH_STATEFULSET_NAMES.split(',') if n.strip()]
        dep_names = [n.strip() for n in OPENSEARCH_DEPLOYMENT_NAMES.split(',') if n.strip()]
        if not sts_names and not dep_names:
            logger.error(
                "Neither OPENSEARCH_STATEFULSET_NAMES nor OPENSEARCH_DEPLOYMENT_NAMES is set; "
                "cannot restart pods"
            )
            return False
        for name in sts_names:
            try:
                result = self._kubectl(['rollout', 'restart', f'statefulset/{name}'])
                if result.returncode != 0:
                    logger.error(f"Failed to restart statefulset/{name}: {result.stderr}")
                    ok = False
                else:
                    logger.info(f"✓ rollout restart triggered for statefulset/{name}")
            except Exception as e:
                logger.error(f"Error restarting statefulset/{name}: {e}")
                ok = False
        for name in dep_names:
            try:
                result = self._kubectl(['rollout', 'restart', f'deployment/{name}'])
                if result.returncode != 0:
                    logger.error(f"Failed to restart deployment/{name}: {result.stderr}")
                    ok = False
                else:
                    logger.info(f"✓ rollout restart triggered for deployment/{name}")
            except Exception as e:
                logger.error(f"Error restarting deployment/{name}: {e}")
                ok = False
        return ok

    # ------------------------------------------------------------------
    # Cluster readiness helpers
    # ------------------------------------------------------------------

    def _wait_for_cluster_ready(self, use_auth: bool,
                                timeout: int = CLUSTER_READY_TIMEOUT) -> bool:
        """
        Poll until the cluster reports green health AND the root API responds.
        When *use_auth* is False the security plugin is disabled and no
        credentials are required.  When True, authenticate with the configured
        admin user.
        """
        session = requests.Session()
        session.verify = False
        requests.packages.urllib3.disable_warnings()
        if use_auth:
            session.auth = HTTPBasicAuth(ES_USERNAME, ES_PASSWORD)

        auth_label = "with auth" if use_auth else "no auth (security disabled)"
        logger.info(f"Waiting for cluster green + API ready ({auth_label}), "
                    f"timeout={timeout}s …")

        deadline = time.time() + timeout
        attempt = 0
        while time.time() < deadline:
            attempt += 1
            try:
                # Wait up to 5 s on the server side for green before returning
                health_url = (f"{OPENSEARCH_ENDPOINT}/_cluster/health"
                              f"?wait_for_status=green&timeout=5s")
                r = session.get(health_url, timeout=15)
                if r.status_code == 200 and r.json().get('status') == 'green':
                    # Additionally verify the root endpoint
                    r2 = session.get(OPENSEARCH_ENDPOINT, timeout=10)
                    if r2.status_code == 200:
                        logger.info(f"✓ Cluster is green and API is reachable "
                                    f"(attempt {attempt})")
                        return True
            except Exception as e:
                logger.debug(f"Cluster not ready yet (attempt {attempt}): {e}")
            time.sleep(CLUSTER_READY_INTERVAL)

        logger.error(f"Timeout ({timeout}s) waiting for cluster to become ready")
        return False

    # ------------------------------------------------------------------
    # Index delete (without auth — called while security is disabled)
    # ------------------------------------------------------------------

    def _delete_security_index(self) -> bool:
        """DELETE .opendistro_security index while the security plugin is disabled."""
        session = requests.Session()
        session.verify = False
        requests.packages.urllib3.disable_warnings()
        url = f"{OPENSEARCH_ENDPOINT}/{OPENDISTRO_SECURITY_INDEX}"
        try:
            r = session.delete(url, timeout=60)
            if r.status_code in (200, 404):
                logger.info(f"✓ Index '{OPENDISTRO_SECURITY_INDEX}' deleted "
                            f"(HTTP {r.status_code})")
                return True
            logger.error(f"Failed to delete '{OPENDISTRO_SECURITY_INDEX}': "
                         f"HTTP {r.status_code} — {r.text}")
            return False
        except Exception as e:
            logger.error(f"Failed to delete '{OPENDISTRO_SECURITY_INDEX}': {e}")
            return False

    # ------------------------------------------------------------------
    # Main orchestrator
    # ------------------------------------------------------------------

    def reinitialize(self) -> bool:
        """Execute the full 7-step security reinitialization flow."""
        if not OPENSEARCH_CONFIG_SECRET_NAME or not OPENSEARCH_NAMESPACE:
            logger.error(
                "OPENSEARCH_CONFIG_SECRET_NAME and OPENSEARCH_NAMESPACE are required "
                "for security reinitialization via disable"
            )
            return False

        logger.info("Starting security reinitialization (via plugins.security.disabled)")

        # 1 ── disable security plugin in config secret
        logger.info("[1/7] Adding plugins.security.disabled: true to opensearch-config secret")
        if not self._add_security_disabled():
            return False

        # 2 ── restart all pods so they pick up the new config
        logger.info("[2/7] Restarting OpenSearch pods (security disabled)")
        if not self._restart_opensearch_workloads():
            return False

        # 3 ── wait for cluster green (no auth needed while security is off)
        logger.info("[3/7] Waiting for cluster green + API (no auth)")
        if not self._wait_for_cluster_ready(use_auth=False):
            return False

        # 4 ── delete the stale security index
        logger.info(f"[4/7] Deleting '{OPENDISTRO_SECURITY_INDEX}' index")
        if not self._delete_security_index():
            return False

        # 5 ── re-enable security plugin
        logger.info("[5/7] Removing plugins.security.disabled from opensearch-config secret")
        if not self._remove_security_disabled():
            return False

        # 6 ── restart again so pods come up with security enabled
        logger.info("[6/7] Restarting OpenSearch pods (security re-enabled)")
        if not self._restart_opensearch_workloads():
            return False

        # 7 ── wait for cluster green (now with auth)
        logger.info("[7/7] Waiting for cluster green + API (with auth)")
        if not self._wait_for_cluster_ready(use_auth=True):
            return False

        logger.info("✓ Security reinitialization completed successfully")
        return True


class BackupManager:
    """Handles backup creation before migration"""
    
    BACKUP_TIMEOUT = 1800  # 30 minutes in seconds
    BACKUP_CHECK_INTERVAL = 10  # Check every 10 seconds
    
    def __init__(self):
        self.backup_url = BACKUP_DAEMON_URL.rstrip('/') if BACKUP_DAEMON_URL else ''
        self.username = BACKUP_DAEMON_API_CREDENTIALS_USERNAME
        self.password = BACKUP_DAEMON_API_CREDENTIALS_PASSWORD
        self.session = requests.Session()
        if self.username and self.password:
            self.session.auth = HTTPBasicAuth(self.username, self.password)
        self.session.verify = False
        requests.packages.urllib3.disable_warnings()
    
    def create_backup(self) -> Optional[str]:
        """
        Create a backup before migration and wait for completion.
        Returns backup ID if successful, None otherwise.
        """
        if not self.backup_url or self.backup_url == 'http://localhost:8080':
            # Check if we're actually running from curator pod (has credentials)
            if not self.username or not self.password:
                logger.info("Backup daemon not configured - skipping backup")
                return None
        
        logger.info("Creating backup before migration...")
        
        try:
            # Step 1: Trigger backup creation
            url = f"{self.backup_url}/backup"
            
            logger.info(f"Sending backup request to: {url}")
            response = self.session.post(url, timeout=60)
            response.raise_for_status()
            
            # Backup ID is returned as plain text
            backup_id = response.text.strip()
            
            if not backup_id:
                logger.error("Backup request returned empty backup ID")
                return None
            
            logger.info(f"Backup started - Backup ID: {backup_id}")
            
            # Step 2: Wait for backup to complete
            if not self._wait_for_backup_completion(backup_id):
                logger.error(f"Backup {backup_id} did not complete successfully")
                return None
            
            logger.info(f"✓ Backup completed successfully - Backup ID: {backup_id}")
            return backup_id
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to create backup: {str(e)}")
            if hasattr(e, 'response') and e.response is not None:
                logger.error(f"Response status: {e.response.status_code}")
                logger.error(f"Response body: {e.response.text}")
            return None
        except Exception as e:
            logger.error(f"Unexpected error during backup: {str(e)}")
            return None
    
    def _wait_for_backup_completion(self, backup_id: str) -> bool:
        """
        Wait for backup to complete successfully.
        Returns True if backup completed successfully, False otherwise.
        """
        logger.info(f"Waiting for backup {backup_id} to complete (timeout: {self.BACKUP_TIMEOUT}s)...")
        
        start_time = time.time()
        check_count = 0
        
        while time.time() - start_time < self.BACKUP_TIMEOUT:
            check_count += 1
            elapsed = int(time.time() - start_time)
            
            try:
                # Check backup status
                status = self._get_backup_status(backup_id)
                
                if status is None:
                    logger.warning(f"Could not get status for backup {backup_id} (check #{check_count})")
                    time.sleep(self.BACKUP_CHECK_INTERVAL)
                    continue
                
                # Check if backup is valid and not failed
                is_valid = status.get('valid', False)
                is_failed = status.get('failed', False)
                exit_code = status.get('exit_code', -1)
                spent_time = status.get('spent_time', 'unknown')
                
                logger.info(
                    f"Backup status (check #{check_count}, elapsed {elapsed}s): "
                    f"valid={is_valid}, failed={is_failed}, exit_code={exit_code}, spent_time={spent_time}"
                )
                
                # Check completion conditions
                if is_valid and not is_failed:
                    logger.info(f"✓ Backup {backup_id} completed successfully after {elapsed}s")
                    return True
                
                if is_failed:
                    logger.error(f"✗ Backup {backup_id} failed: {status}")
                    return False
                
                # Backup still in progress, wait before next check
                time.sleep(self.BACKUP_CHECK_INTERVAL)
                
            except Exception as e:
                logger.warning(f"Error checking backup status (check #{check_count}): {str(e)}")
                time.sleep(self.BACKUP_CHECK_INTERVAL)
        
        # Timeout reached
        logger.error(f"✗ Timeout waiting for backup {backup_id} to complete (>{self.BACKUP_TIMEOUT}s)")
        return False
    
    def _get_backup_status(self, backup_id: str) -> Optional[Dict]:
        """
        Get backup status from daemon.
        Returns status dict or None if request fails.
        """
        try:
            url = f"{self.backup_url}/listbackups/{backup_id}"
            response = self.session.get(url, timeout=10)
            response.raise_for_status()
            
            return response.json()
            
        except requests.exceptions.RequestException as e:
            logger.debug(f"Failed to get backup status: {str(e)}")
            return None
        except Exception as e:
            logger.debug(f"Error parsing backup status: {str(e)}")
            return None


class DBaaSUserRestorer:
    """Handles DBaaS user password restoration"""
    
    def __init__(self):
        self.adapter_endpoint = DBAAS_ADAPTER_ADDRESS.rstrip('/') if DBAAS_ADAPTER_ADDRESS else ''
        self.adapter_username = DBAAS_ADAPTER_USERNAME
        self.adapter_password = DBAAS_ADAPTER_PASSWORD
        self.session = requests.Session()
        if self.adapter_username and self.adapter_password:
            self.session.auth = HTTPBasicAuth(self.adapter_username, self.adapter_password)
        self.session.verify = False
        requests.packages.urllib3.disable_warnings()
    
    def restore_users(self) -> bool:
        """Restore user passwords via DBaaS adapter API"""
        # Check if DBaaS is enabled by checking if adapter address is configured
        if not self.adapter_endpoint:
            logger.info("DBaaS adapter not configured - skipping user restoration")
            return True
        
        logger.info("Starting DBaaS user restoration...")
        
        try:
            # Assume v2 API (matching dbaasAdapter.apiVersion default)
            api_version = 'v2'
            
            # Check current state
            state = self._get_restore_state(api_version)
            
            if state != USERS_RECOVERY_RUNNING_STATE:
                state = USERS_RECOVERY_IDLE_STATE
            
            restore_failed = False
            start_time = time.time()
            
            while state not in [USERS_RECOVERY_DONE_STATE, USERS_RECOVERY_FAILED_STATE]:
                if time.time() - start_time > DBAAS_TIMEOUT:
                    logger.error(f"Timeout reached during user restoration (>{DBAAS_TIMEOUT}s)")
                    return False
                
                if state == USERS_RECOVERY_IDLE_STATE:
                    # Trigger restoration via DBaaS adapter
                    if not self._trigger_restore(api_version):
                        restore_failed = True
                        break
                
                # Wait and check state
                time.sleep(DBAAS_INTERVAL)
                state = self._get_restore_state(api_version)
            
            if state == USERS_RECOVERY_FAILED_STATE or restore_failed:
                logger.error(f"User restoration failed with state: {state}")
                return False
            
            logger.info(f"✓ User restoration completed successfully (state: {state})")
            return True
            
        except Exception as e:
            logger.error(f"User restoration failed: {str(e)}")
            return False
    
    def _get_restore_state(self, api_version: str) -> str:
        """Get current restoration state from adapter"""
        try:
            url = f"{self.adapter_endpoint}/api/{api_version}/dbaas/adapter/opensearch/users/restore-password/state"
            
            response = self.session.get(url, timeout=30)
            response.raise_for_status()
            return response.text.strip()
        except Exception as e:
            logger.warning(f"Could not get restore state: {str(e)}")
            return USERS_RECOVERY_IDLE_STATE
    
    def _trigger_restore(self, api_version: str) -> bool:
        """Trigger user password restoration via DBaaS adapter"""
        try:
            url = f"{self.adapter_endpoint}/api/{api_version}/dbaas/adapter/opensearch/users/restore-password"
            
            response = self.session.post(url, json={}, timeout=30)
            response.raise_for_status()
            
            logger.info("User restoration triggered successfully")
            return True
            
        except Exception as e:
            logger.error(f"Failed to trigger user restoration: {str(e)}")
            return False


def main():
    """Main migration workflow"""
    parser = argparse.ArgumentParser(
        description='Migrate OpenSearch 1.x indices to 2.x/3.x format'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Perform a dry run without making changes'
    )
    parser.add_argument(
        '--skip-space-check',
        action='store_true',
        help='Skip disk space validation (not recommended)'
    )
    parser.add_argument(
        '--skip-security-reinit',
        action='store_true',
        help='Skip security reinitialization (disable plugin, delete index, re-enable)'
    )
    parser.add_argument(
        '--skip-dbaas-restore',
        action='store_true',
        help='Skip DBaaS user restoration'
    )
    parser.add_argument(
        '--skip-backup',
        action='store_true',
        help='Skip creating backup before migration'
    )
    
    args = parser.parse_args()
    
    logger.info("=" * 80)
    logger.info("OpenSearch 1.x Index Migration Script")
    logger.info("=" * 80)
    logger.info(f"OpenSearch Endpoint: {OPENSEARCH_ENDPOINT}")
    logger.info(f"OpenSearch Username: {ES_USERNAME}")
    logger.info(f"Migration Mode: {MIGRATION_MODE}")
    logger.info(f"Dry Run: {args.dry_run}")
    logger.info("=" * 80)
    
    try:
        # Initialize client
        client = OpenSearchClient(OPENSEARCH_ENDPOINT, ES_USERNAME, ES_PASSWORD)
        
        # Verify connection and get version
        cluster_info = client.request('GET', '/')
        cluster_data = cluster_info.json()
        logger.info(f"Connected to OpenSearch cluster: {cluster_data.get('cluster_name')}")
        
        current_version = cluster_data.get('version', {}).get('number', 'unknown')
        logger.info(f"Current OpenSearch version: {current_version}")
        
        # Initialize migrator
        migrator = IndexMigrator(client)
        
        # Extract major version from current version string
        try:
            current_major = int(current_version.split('.')[0])
        except:
            current_major = 0
        
        logger.info(f"Current major version: {current_major}")
        
        # Step 1: Detect 1.x indices
        indices_to_migrate, security_needs_reinit = migrator.get_1x_indices()
        
        if not indices_to_migrate and not security_needs_reinit:
            logger.info("No OpenSearch 1.x indices found - migration not needed")
            logger.info("Migration completed successfully")
            return 0
        
        if indices_to_migrate:
            logger.info(f"Found {len(indices_to_migrate)} indices requiring migration:")
        
        if security_needs_reinit:
            logger.info("Security index (.opendistro_security) needs reinitialization")
        
        # PRE-DEPLOY CHECK MODE: Check if running on OpenSearch 3.x with legacy indices
        if MIGRATION_MODE == 'pre-deploy-check':
            logger.warning("=" * 80)
            logger.warning("PRE-DEPLOY MIGRATION CHECK MODE")
            logger.warning("=" * 80)
            
            # Check if this is OpenSearch 3.x with legacy 1.x indices
            if current_major >= 3 and (indices_to_migrate or security_needs_reinit):
                logger.error("=" * 80)
                logger.error("MIGRATION REQUIRED BEFORE UPGRADE TO 3.x")
                logger.error("=" * 80)
                logger.error("")
                if indices_to_migrate:
                    logger.error(f"Found {len(indices_to_migrate)} legacy indices created in OpenSearch 1.x")
                    logger.error(f"Indices: {', '.join(indices_to_migrate)}")
                if security_needs_reinit:
                    logger.error("Security index (.opendistro_security) was created in OpenSearch 1.x")
                    logger.error("Security will be reinitialized automatically")
                logger.error("")
                logger.error("These indices MUST be migrated before upgrading to OpenSearch 3.x")
                logger.error("")
                logger.error("RECOMMENDED STEPS:")
                logger.error("1. Rollback to OpenSearch 2.x")
                logger.error("2. Perform manual migration of legacy indices:")
                logger.error("   kubectl exec -it <curator-pod> -n <namespace> -- /bin/bash")
                logger.error("   python3 /opt/elasticsearch-curator/migrate_opensearch_1x_indices.py")
                logger.error("3. Verify migration completed successfully")
                logger.error("4. Then upgrade to OpenSearch 3.x")
                logger.error("")
                logger.error("ALTERNATIVE:")
                logger.error("Enable automatic migration in values.yaml:")
                logger.error("  migration:")
                logger.error("    enabled: true")
                logger.error("")
                logger.error("For detailed instructions, see: docs/public/migration-indices.md")
                logger.error("=" * 80)
                return 3  # Exit code 3 = migration required before upgrade
            elif current_major < 3:
                logger.info(f"Running on OpenSearch {current_major}.x")
                if indices_to_migrate or security_needs_reinit:
                    logger.warning("Legacy 1.x indices detected but running on 2.x - migration recommended before upgrading to 3.x")
                logger.info("Pre-deploy check passed")
                return 0
            else:
                logger.info(f"Running on OpenSearch {current_major}.x")
                logger.info("No legacy 1.x indices detected")
                logger.info("Pre-deploy check passed")
                return 0
        
        # NORMAL MIGRATION MODE (manual or automatic)
        if args.dry_run:
            logger.info("DRY RUN MODE - No changes will be made")
            if indices_to_migrate:
                logger.info(f"Indices that would be migrated: {indices_to_migrate}")
            if security_needs_reinit:
                logger.info("Security would be reinitialized")
            return 0
        
        # Step 1: Create backup before migration
        backup_id = None
        if not args.skip_backup:
            logger.info("")
            logger.info("=" * 80)
            logger.info("STEP 1: CREATING BACKUP BEFORE MIGRATION")
            logger.info("=" * 80)
            backup_mgr = BackupManager()
            backup_id = backup_mgr.create_backup()
            if backup_id is None:
                logger.error("=" * 80)
                logger.error("Backup creation or validation failed - cannot proceed with migration")
                logger.error("This ensures you have a recovery point if migration encounters issues")
                logger.error("")
                logger.error("Possible reasons:")
                logger.error("  - Backup daemon is not accessible")
                logger.error("  - Backup process failed or timed out (30 minute timeout)")
                logger.error("  - Backup validation failed (valid=false or failed=true)")
                logger.error("")
                logger.error("To bypass this check (NOT RECOMMENDED):")
                logger.error("  Use --skip-backup flag")
                logger.error("=" * 80)
                return 2
            logger.info(f"✓ Backup validated successfully - ID: {backup_id}")
        else:
            logger.warning("Skipping backup creation (--skip-backup)")
        
        # Step 2: Validate disk space (only if there are indices to migrate)
        if indices_to_migrate:
            logger.info("")
            logger.info("=" * 80)
            logger.info("STEP 2: VALIDATING DISK SPACE")
            logger.info("=" * 80)
            if not args.skip_space_check:
                migrator.validate_disk_space(indices_to_migrate)
            else:
                logger.warning("Skipping disk space validation (--skip-space-check)")
        
        # Step 3: Migrate indices sequentially (only if there are indices)
        if indices_to_migrate:
            logger.info("")
            logger.info("=" * 80)
            logger.info(f"STEP 3: MIGRATING {len(indices_to_migrate)} INDICES")
            logger.info("=" * 80)
            logger.info("")
            
            for i, index in enumerate(indices_to_migrate, 1):
                logger.info(f"\n{'=' * 80}")
                logger.info(f"MIGRATING INDEX {i}/{len(indices_to_migrate)}: {index}")
                logger.info(f"{'=' * 80}")
                is_system_index = index.startswith('.')
                migrator.migrate_index(index, is_system_index=is_system_index)
        else:
            logger.info("")
            logger.info("=" * 80)
            logger.info("STEP 3: NO INDICES TO MIGRATE")
            logger.info("=" * 80)
        
        # Step 4: Reinitialize security (always do this if security_needs_reinit OR if we migrated indices)
        if security_needs_reinit or indices_to_migrate:
            logger.info("")
            logger.info("=" * 80)
            logger.info("STEP 4: REINITIALIZING SECURITY CONFIGURATION")
            logger.info("=" * 80)
            if security_needs_reinit:
                logger.info("Security index was created in 1.x - reinitialization REQUIRED")
            if not args.skip_security_reinit:
                reinit_mgr = SecurityReinitViaDisable(client)
                if not reinit_mgr.reinitialize():
                    logger.error("Security reinitialization failed - this is a critical error!")
                    if security_needs_reinit:
                        logger.error("Security index MUST be reinitialized for OpenSearch 3.x compatibility")
                        return 4  # Exit code 4 = security reinit failed
                    else:
                        logger.warning("Manual intervention may be required")
                else:
                    logger.info("Security reinitialization completed successfully")
            else:
                logger.warning("Skipping security reinitialization (--skip-security-reinit)")
                if security_needs_reinit:
                    logger.error("WARNING: Security index needs reinitialization but it was skipped!")
        
        # Step 5: Restore DBaaS users
        logger.info("")
        logger.info("=" * 80)
        logger.info("STEP 5: RESTORING DBAAS USERS")
        logger.info("=" * 80)
        if not args.skip_dbaas_restore:
            dbaas_restorer = DBaaSUserRestorer()
            if not dbaas_restorer.restore_users():
                logger.warning("DBaaS user restoration failed - users may need to be recreated manually")
            else:
                logger.info("DBaaS user restoration completed successfully")
        else:
            logger.warning("Skipping DBaaS user restoration (--skip-dbaas-restore)")
        
        # Summary
        logger.info("")
        logger.info("=" * 80)
        logger.info("MIGRATION SUMMARY")
        logger.info("=" * 80)
        if backup_id:
            logger.info(f"Backup ID: {backup_id}")
        if indices_to_migrate:
            logger.info(f"Total indices migrated: {len(indices_to_migrate)}")
        if security_needs_reinit:
            logger.info("Security reinitialized: YES (required for 1.x security index)")
        logger.info("Migration completed successfully")
        logger.info("=" * 80)
        return 0
        
    except InsufficientSpaceError as e:
        logger.error(f"✗ Migration aborted: {str(e)}")
        return 2
    except MigrationError as e:
        logger.error(f"✗ Migration failed: {str(e)}")
        return 1
    except Exception as e:
        logger.exception(f"✗ Unexpected error during migration: {str(e)}")
        return 1


if __name__ == '__main__':
    sys.exit(main())
