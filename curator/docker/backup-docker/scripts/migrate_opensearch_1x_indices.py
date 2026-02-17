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
import json
import logging
import os
import sys
import time
from typing import Dict, List, Tuple, Optional
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

# Pre-deploy job configuration
MIGRATION_MODE = os.environ.get('MIGRATION_MODE', 'manual')  # 'manual', 'migration', or 'pre-deploy-check'
TARGET_OPENSEARCH_VERSION = os.environ.get('TARGET_OPENSEARCH_VERSION', '')  # Target version for upgrade
CURRENT_OPENSEARCH_VERSION = os.environ.get('CURRENT_OPENSEARCH_VERSION', '')  # Current version from StatefulSet
OPENSEARCH_SECURITY_ADMIN_PATH = os.environ.get('OPENSEARCH_SECURITY_ADMIN_PATH', 
                                                '/usr/share/opensearch/plugins/opensearch-security/tools/securityadmin.sh')
OPENSEARCH_SECURITY_CONFIG_PATH = os.environ.get('OPENSEARCH_SECURITY_CONFIG_PATH',
                                                 '/usr/share/opensearch/config/opensearch-security')
OPENSEARCH_CONFIG_PATH = os.environ.get('OPENSEARCH_CONFIG_PATH', '/usr/share/opensearch/config')
OPENSEARCH_POD_NAME = os.environ.get('OPENSEARCH_POD_NAME', '')
OPENSEARCH_NAMESPACE = os.environ.get('OPENSEARCH_NAMESPACE', 'default')

# Constants
MIGRATION_SUFFIX = '-migration'
USERS_RECOVERY_DONE_STATE = 'done'
USERS_RECOVERY_FAILED_STATE = 'failed'
USERS_RECOVERY_RUNNING_STATE = 'running'
USERS_RECOVERY_IDLE_STATE = 'idle'
DBAAS_TIMEOUT = 240
DBAAS_INTERVAL = 10

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
    
    def get_cluster_info(self) -> Dict:
        """Get cluster information"""
        logger.info("Fetching cluster information...")
        response = self.client.get('/')
        return response.json()
    
    def get_current_opensearch_version(self) -> str:
        """Get currently running OpenSearch version"""
        try:
            cluster_info = self.get_cluster_info()
            version = cluster_info.get('version', {}).get('number', 'unknown')
            logger.info(f"Current OpenSearch version: {version}")
            return version
        except Exception as e:
            logger.error(f"Could not determine OpenSearch version: {str(e)}")
            return 'unknown'
    
    def get_major_version(self, version_string: str) -> int:
        """Extract major version number from version string (e.g., '2.11.0' -> 2)"""
        try:
            return int(version_string.split('.')[0])
        except:
            return 0
    
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
        """Check if an index was created in OpenSearch 1.x"""
        try:
            settings = self.get_index_settings(index)
            index_settings = settings.get(index, {}).get('settings', {}).get('index', {})
            
            # Check version in settings
            version_created = index_settings.get('version', {}).get('created', '')
            if version_created:
                # Version format: 135217827 for 1.x, 136217827 for 2.x, 137xxxxxx for 3.x
                # First digit(s) represent major version
                version_str = str(version_created)
                if len(version_str) >= 9:
                    major_version = int(version_str[0:2]) if version_str[0] == '1' else int(version_str[0])
                    if major_version <= 13 or (major_version >= 130 and major_version < 136):
                        logger.info(f"Index '{index}' was created in OpenSearch 1.x (version: {version_created})")
                        return True
            
            # Alternative: check creation date and OpenSearch version history
            # If the cluster was upgraded from 1.x, indices from that time are 1.x indices
            
            return False
        except Exception as e:
            logger.warning(f"Could not determine version for index '{index}': {str(e)}")
            return False
    
    def get_1x_indices(self) -> List[str]:
        """Get all indices created in OpenSearch 1.x"""
        logger.info("Detecting indices created in OpenSearch 1.x...")
        all_indices = self.get_all_indices()
        
        indices_1x = []
        for idx_info in all_indices:
            index = idx_info.get('index', '')
            
            # Skip system indices
            if index.startswith('.'):
                continue
            
            # Skip migration indices
            if index.endswith(MIGRATION_SUFFIX):
                continue
            
            if self.is_1x_index(index):
                indices_1x.append(index)
        
        logger.info(f"Found {len(indices_1x)} indices created in OpenSearch 1.x: {indices_1x}")
        return indices_1x
    
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
                
                logger.info(f"✓ Reindexing completed: {created}/{total} documents")
                
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
    
    def migrate_index(self, index: str):
        """
        Migrate a single index from 1.x to 2.x/3.x format
        
        Process:
        1. Close the index
        2. Get mappings from original index
        3. Reindex to temporary index with migration suffix
        4. Delete original index
        5. Reindex back to original name (creates with new format)
        6. Delete temporary index
        7. Open the index
        """
        logger.info(f"=" * 80)
        logger.info(f"Starting migration for index: '{index}'")
        logger.info(f"=" * 80)
        
        migration_index = f"{index}{MIGRATION_SUFFIX}"
        
        try:
            # Step 1: Get mappings and settings before closing
            logger.info(f"Step 1: Fetching mappings and settings for '{index}'...")
            mappings_response = self.get_index_mappings(index)
            mappings = mappings_response.get(index, {}).get('mappings', {})
            
            settings_response = self.get_index_settings(index)
            index_settings = settings_response.get(index, {}).get('settings', {}).get('index', {})
            
            # Extract relevant settings (exclude version-specific ones)
            preserved_settings = {}
            for key in ['number_of_shards', 'number_of_replicas', 'refresh_interval']:
                if key in index_settings:
                    preserved_settings[key] = index_settings[key]
            
            logger.info(f"✓ Retrieved mappings and settings")
            
            # Step 2: Check if migration index already exists (idempotency)
            if self.index_exists(migration_index):
                logger.warning(f"Migration index '{migration_index}' already exists - cleaning up from previous run")
                self.delete_index(migration_index)
            
            # Step 3: Reindex to migration index
            logger.info(f"Step 2: Reindexing '{index}' → '{migration_index}'...")
            self.reindex(index, migration_index)
            
            # Step 4: Close original index
            logger.info(f"Step 3: Closing original index '{index}'...")
            self.close_index(index)
            
            # Step 5: Delete original index
            logger.info(f"Step 4: Deleting original index '{index}'...")
            self.delete_index(index)
            
            # Step 6: Create new index with proper mappings (if mappings exist)
            if mappings:
                logger.info(f"Step 5: Creating '{index}' with preserved mappings...")
                # Remove any version-specific fields from mappings
                clean_mappings = self._clean_mappings(mappings)
                self.create_index_with_mappings(index, clean_mappings, {'index': preserved_settings} if preserved_settings else None)
            
            # Step 7: Reindex back to original name
            logger.info(f"Step 6: Reindexing '{migration_index}' → '{index}'...")
            self.reindex(migration_index, index)
            
            # Step 8: Delete migration index
            logger.info(f"Step 7: Deleting migration index '{migration_index}'...")
            self.delete_index(migration_index)
            
            # Step 9: Open the index
            logger.info(f"Step 8: Opening migrated index '{index}'...")
            self.open_index(index)
            
            logger.info(f"✓ Successfully migrated index '{index}'")
            logger.info(f"=" * 80)
            
        except Exception as e:
            logger.error(f"✗ Migration failed for index '{index}': {str(e)}")
            # Attempt cleanup
            self._cleanup_failed_migration(index, migration_index)
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
    
    def _cleanup_failed_migration(self, original: str, migration: str):
        """Attempt to cleanup after a failed migration"""
        logger.info("Attempting cleanup after failed migration...")
        
        try:
            # If migration index exists, we can potentially restore
            if self.index_exists(migration):
                logger.info(f"Migration index '{migration}' exists")
                
                if not self.index_exists(original):
                    logger.info(f"Original index '{original}' missing - restoring from migration index")
                    # Restore from migration index
                    self.reindex(migration, original)
                    self.delete_index(migration)
                    self.open_index(original)
                    logger.info("✓ Restored original index from migration backup")
                else:
                    logger.info(f"Original index '{original}' exists - just cleaning up migration index")
                    self.delete_index(migration)
            else:
                logger.warning("Migration index does not exist - cannot restore")
                
        except Exception as e:
            logger.error(f"Cleanup failed: {str(e)}")
            logger.error("Manual intervention may be required!")


class SecurityManager:
    """Manages OpenSearch security configuration"""
    
    def __init__(self, client: OpenSearchClient):
        self.client = client
    
    def backup_security_config(self) -> bool:
        """Backup security configuration using securityadmin tool via kubectl exec"""
        logger.info("Backing up OpenSearch security configuration...")
        
        if not OPENSEARCH_POD_NAME:
            logger.warning("OPENSEARCH_POD_NAME not set - skipping security backup")
            return False
        
        try:
            backup_dir = f"{OPENSEARCH_SECURITY_CONFIG_PATH}/migration-backup"
            
            # Create backup directory
            cmd_mkdir = [
                'kubectl', 'exec', '-n', OPENSEARCH_NAMESPACE, OPENSEARCH_POD_NAME, 
                '--', 'mkdir', '-p', backup_dir
            ]
            subprocess.run(cmd_mkdir, check=True, capture_output=True)
            
            # Run securityadmin backup
            cmd_backup = [
                'kubectl', 'exec', '-n', OPENSEARCH_NAMESPACE, OPENSEARCH_POD_NAME, '--',
                OPENSEARCH_SECURITY_ADMIN_PATH,
                '-backup', backup_dir,
                '-cert', f'{OPENSEARCH_CONFIG_PATH}/admin-crt.pem',
                '-cacert', f'{OPENSEARCH_CONFIG_PATH}/admin-root-ca.pem',
                '-key', f'{OPENSEARCH_CONFIG_PATH}/admin-key.pem',
                '-h', 'localhost'
            ]
            
            result = subprocess.run(cmd_backup, check=True, capture_output=True, text=True)
            logger.info(f"✓ Security configuration backed up to {backup_dir}")
            logger.debug(f"Backup output: {result.stdout}")
            return True
            
        except subprocess.CalledProcessError as e:
            logger.error(f"Failed to backup security configuration: {e.stderr}")
            return False
        except Exception as e:
            logger.error(f"Failed to backup security configuration: {str(e)}")
            return False
    
    def reinitialize_security(self) -> bool:
        """Reinitialize security configuration using securityadmin tool"""
        logger.info("Reinitializing OpenSearch security...")
        
        if not OPENSEARCH_POD_NAME:
            logger.warning("OPENSEARCH_POD_NAME not set - skipping security reinitialization")
            return False
        
        try:
            # Run securityadmin to reinitialize
            cmd = [
                'kubectl', 'exec', '-n', OPENSEARCH_NAMESPACE, OPENSEARCH_POD_NAME, '--',
                OPENSEARCH_SECURITY_ADMIN_PATH,
                '-cd', OPENSEARCH_SECURITY_CONFIG_PATH,
                '-cert', f'{OPENSEARCH_CONFIG_PATH}/admin-crt.pem',
                '-cacert', f'{OPENSEARCH_CONFIG_PATH}/admin-root-ca.pem',
                '-key', f'{OPENSEARCH_CONFIG_PATH}/admin-key.pem',
                '-h', 'localhost',
                '-icl', '-nhnv'
            ]
            
            result = subprocess.run(cmd, check=True, capture_output=True, text=True)
            logger.info("✓ Security configuration reinitialized")
            logger.debug(f"Reinitialization output: {result.stdout}")
            return True
            
        except subprocess.CalledProcessError as e:
            logger.error(f"Failed to reinitialize security: {e.stderr}")
            return False
        except Exception as e:
            logger.error(f"Failed to reinitialize security: {str(e)}")
            return False


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
        '--skip-security-backup',
        action='store_true',
        help='Skip security configuration backup'
    )
    parser.add_argument(
        '--skip-dbaas-restore',
        action='store_true',
        help='Skip DBaaS user restoration'
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
        
        # Get current major version
        current_major = migrator.get_major_version(current_version)
        target_major = migrator.get_major_version(TARGET_OPENSEARCH_VERSION) if TARGET_OPENSEARCH_VERSION else current_major
        
        logger.info(f"Current major version: {current_major}")
        if TARGET_OPENSEARCH_VERSION:
            logger.info(f"Target major version: {target_major}")
        
        # Step 1: Detect 1.x indices
        indices_to_migrate = migrator.get_1x_indices()
        
        if not indices_to_migrate:
            logger.info("No OpenSearch 1.x indices found - migration not needed")
            logger.info("✓ Migration completed successfully")
            return 0
        
        logger.info(f"Found {len(indices_to_migrate)} indices requiring migration")
        
        # PRE-DEPLOY CHECK MODE: Check if upgrading from 2.x to 3.x with legacy indices
        if MIGRATION_MODE == 'pre-deploy-check':
            logger.warning("=" * 80)
            logger.warning("PRE-DEPLOY MIGRATION CHECK MODE")
            logger.warning("=" * 80)
            
            # Check if this is an upgrade from 2.x to 3.x
            is_2x_to_3x_upgrade = (current_major == 2 and target_major == 3)
            
            if is_2x_to_3x_upgrade:
                logger.error("=" * 80)
                logger.error("MIGRATION REQUIRED BEFORE UPGRADE TO 3.x")
                logger.error("=" * 80)
                logger.error("")
                logger.error(f"Found {len(indices_to_migrate)} legacy indices created in OpenSearch 1.x")
                logger.error(f"Indices: {', '.join(indices_to_migrate)}")
                logger.error("")
                logger.error("These indices MUST be migrated before upgrading to OpenSearch 3.x")
                logger.error("")
                logger.error("RECOMMENDED STEPS:")
                logger.error("1. Install the latest OpenSearch 2.x version first")
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
            else:
                logger.info(f"Not a 2.x -> 3.x upgrade (current: {current_major}, target: {target_major})")
                logger.info("Legacy indices detected but upgrade path allows them")
                logger.info("✓ Pre-deploy check passed (upgrade can proceed)")
                return 0
        
        # NORMAL MIGRATION MODE (manual or automatic)
        if args.dry_run:
            logger.info("DRY RUN MODE - No changes will be made")
            logger.info(f"Indices that would be migrated: {indices_to_migrate}")
            return 0
        
        # Step 2: Validate disk space
        if not args.skip_space_check:
            migrator.validate_disk_space(indices_to_migrate)
        else:
            logger.warning("Skipping disk space validation (--skip-space-check)")
        
        # Step 3: Backup security configuration
        if not args.skip_security_backup:
            security_mgr = SecurityManager(client)
            security_mgr.backup_security_config()
        else:
            logger.warning("Skipping security backup (--skip-security-backup)")
        
        # Step 4: Migrate indices sequentially
        logger.info(f"Starting sequential migration of {len(indices_to_migrate)} indices...")
        
        failed_indices = []
        for i, index in enumerate(indices_to_migrate, 1):
            logger.info(f"\nMigrating index {i}/{len(indices_to_migrate)}: {index}")
            try:
                migrator.migrate_index(index)
            except Exception as e:
                logger.error(f"Failed to migrate index '{index}': {str(e)}")
                failed_indices.append(index)
                # Continue with next index instead of failing completely
        
        # Step 5: Reinitialize security
        if not args.skip_security_backup:
            logger.info("\nReinitializing security configuration...")
            security_mgr = SecurityManager(client)
            if not security_mgr.reinitialize_security():
                logger.warning("Security reinitialization failed - manual intervention may be required")
        
        # Step 6: Restore DBaaS users
        if not args.skip_dbaas_restore:
            logger.info("\nRestoring DBaaS users...")
            dbaas_restorer = DBaaSUserRestorer()
            if not dbaas_restorer.restore_users():
                logger.warning("DBaaS user restoration failed - users may need to be recreated manually")
        else:
            logger.warning("Skipping DBaaS user restoration (--skip-dbaas-restore)")
        
        # Summary
        logger.info("\n" + "=" * 80)
        logger.info("Migration Summary")
        logger.info("=" * 80)
        logger.info(f"Total indices processed: {len(indices_to_migrate)}")
        logger.info(f"Successfully migrated: {len(indices_to_migrate) - len(failed_indices)}")
        logger.info(f"Failed: {len(failed_indices)}")
        
        if failed_indices:
            logger.error(f"Failed indices: {failed_indices}")
            logger.error("✗ Migration completed with errors")
            return 1
        
        logger.info("✓ Migration completed successfully")
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
